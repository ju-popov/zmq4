# REQ/REP Socket Pattern — Behaviour Across Transport Types

## REQ/REP State Machine

REQ/REP enforces a strict alternating message exchange regardless of transport:

- **REQ** must: `Send` → `Recv` → `Send` → `Recv` → ...
- **REP** must: `Recv` → `Send` → `Recv` → `Send` → ...

Calling `Send` twice in a row, or `Recv` before `Send` on a REQ socket, returns an `EFSM`
error immediately. This is enforced in ZMQ's socket layer, not in the transport.

---

## Transport Types

### `tcp://`

The standard transport for networked and local communication.

**Connect/Bind:**
- `Connect` returns immediately regardless of whether the server is running.
  ZMQ establishes the TCP connection in the background (lazy connect).
- `Bind` can happen before or after `Connect` on the other side.

**Send:**
- Returns immediately and enqueues the message in ZMQ's internal send buffer.
- If the peer is not yet connected, the message is held in the buffer until
  the connection is established and then delivered.
- If the peer has disconnected, ZMQ detects the TCP FIN/RST and **drops** the
  queued message silently. No error is returned to the caller.
- ZMQ automatically attempts to reconnect using exponential backoff.

**Recv:**
- Blocks until a message arrives. There is no built-in timeout; if the peer
  never sends (or has gone away), `Recv` blocks forever unless a socket option
  such as `ZMQ_RCVTIMEO` is set.

---

### `ipc://`

Unix domain socket transport. Semantically identical to `tcp://` for `Send`/`Recv`.

**Differences from `tcp://`:**
- Same-machine only. Endpoint is a filesystem path: `ipc:///tmp/myapp.sock`.
- Slightly lower latency (no IP/TCP stack overhead).
- Disconnect detection and reconnect behavior are the same as `tcp://`.
- Send/Recv blocking and buffering rules are identical.

---

### `inproc://`

In-process transport between threads sharing the same ZMQ context.

**Connect/Bind:**
- `Bind` **must** be called before `Connect`. Unlike `tcp://` and `ipc://`,
  there is no lazy connect — if `Connect` is called before `Bind`, it fails.

**Send:**
- Transfers the message directly through shared memory. No kernel involvement,
  no serialization, no copy through the network stack.
- If the receiving thread has closed its socket, `Send` returns an error
  (`ETERM` or `ENOTSUP`) immediately — there is no silent drop as with `tcp://`,
  because ZMQ can detect the closed socket synchronously within the process.

**Recv:**
- Blocks until the sending thread calls `Send`. Same blocking rules as `tcp://`.
- Because both sides are in the same process, a crashed thread typically brings
  down the whole process, making disconnect scenarios largely moot.

**Latency:**
- Lowest possible — no OS involvement for message transfer.

---

### `pgm://` and `epgm://`

UDP multicast transports. **Not compatible with REQ/REP.**

These transports only work with `PUB`/`SUB` because multicast is inherently
one-to-many and connectionless — there is no concept of a peer to reply to.
Using `pgm://` or `epgm://` with a REQ or REP socket returns a bind/connect error.

---

## High-Water Mark (HWM)

ZMQ uses two per-socket buffer limits to bound memory usage:

| Option | Direction | Default |
|---|---|---|
| `ZMQ_SNDHWM` | outgoing messages | 1000 |
| `ZMQ_RCVHWM` | incoming messages | 1000 |

These limits apply to the number of messages queued in ZMQ's internal buffers,
not bytes. They are independent of transport type.

**What happens when the send HWM is reached:**

- With blocking `Send` (flag `0`): `Send` blocks until the peer consumes a
  message and space becomes available. Memory does not grow beyond the limit.
- With non-blocking `Send` (flag `DONTWAIT`): `Send` returns immediately with
  an `EAGAIN` error. The message is discarded by the caller, not by ZMQ.

**HWM in practice for REQ/REP:**

Because REQ enforces a strict send→recv cycle, a REQ socket can have at most
one message outstanding at any time. The send HWM of 1000 is therefore never
reached in normal REQ/REP operation. It would only become relevant if the
application holds many REQ sockets simultaneously, each waiting for a reply.

---

## What Happens to a Sent Message When the Other Side Is Gone

The outcome depends on **when** the peer disappears relative to the `Send` call.

### Peer gone before `Send` (never connected, or disconnected earlier)

| Transport | Outcome |
|---|---|
| `tcp://` | Message is queued in the send buffer. ZMQ retries the connection. If the peer comes back before HWM is reached, the message is delivered. If HWM is reached first, `Send` blocks (blocking mode) or returns `EAGAIN` (non-blocking). |
| `ipc://` | Same as `tcp://`. |
| `inproc://` | `Send` returns an error immediately (`ETERM`). The message is never queued. |

### Peer gone after `Send` but before it consumed the message

| Transport | Outcome |
|---|---|
| `tcp://` | ZMQ detects the TCP disconnect and **silently drops** the queued message. No error is returned to the caller — `Send` already returned successfully. |
| `ipc://` | Same as `tcp://`. |
| `inproc://` | Same process — peer socket closure is detected immediately. Subsequent `Send` calls return an error, but a message already in the buffer at the moment of closure may be dropped. |

### REP-specific case: client gone during server work

The server calls `Recv` (succeeds), does work, then calls `Send` — but the
client disconnected while the server was working. `Send` succeeds and returns
no error. ZMQ queues the reply briefly and then drops it when it detects the
TCP peer is gone. This is **not a memory leak**: the allocation is temporary
and bounded by `ZMQ_SNDHWM`. The REP state machine advances normally to
waiting for the next request.

### REQ-specific case: server gone after client `Send`

The client calls `Send` (succeeds, message buffered), then calls `Recv` — but
the server never processes the request. `Recv` blocks forever. ZMQ will
reconnect to the server when it comes back, but the buffered request is
**not automatically retransmitted** after reconnect — it was already consumed
from the REQ socket's buffer. The client remains stuck in `Recv` unless a
`ZMQ_RCVTIMEO` timeout is configured.
