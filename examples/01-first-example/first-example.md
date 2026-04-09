# ZMQ REQ/REP Send/Receive Behaviour by Transport

## High-Water Mark (HWM)

ZMQ uses per-socket buffer limits to bound memory usage.

| Option | Direction | Default |
|---|---|---|
| `ZMQ_SNDHWM` | outgoing messages (per peer queue) | 1000 |
| `ZMQ_RCVHWM` | incoming messages | 1000 |

Limits are counted in **number of messages**, not bytes. What happens when a
limit is reached depends on the socket type and is described in each scenario below.

---

## Scenario 1: REQ — Send — TCP

The REQ socket enforces a strict alternating state machine: `Send` → `Recv` → `Send` → ...
Calling `Send` twice in a row returns `EFSM` immediately.

`Send` returns immediately and enqueues the message in ZMQ's internal send buffer.
The actual TCP transmission happens in the background. `Connect` always returns
successfully regardless of whether the server is running (lazy connect) — if the
REP server is not yet connected, the message is held in the buffer and delivered
once the connection is established.

**If the REP server is gone before `Send`:** the message is queued in the send
buffer. ZMQ retries the connection using exponential backoff. The message is
delivered if the server comes back before HWM is reached.

**If the REP server goes away after `Send` but before processing the message:**
ZMQ detects the TCP disconnect (FIN/RST) and silently drops the queued message.
`Send` already returned successfully — no error is reported. The REQ socket
remains stuck waiting in `Recv` (see Scenario 2).

**If HWM (`ZMQ_SNDHWM`) is reached:** `Send` blocks until the peer consumes a
message and buffer space becomes available. With the non-blocking flag
(`DONTWAIT`), `Send` returns `EAGAIN` instead. In practice, REQ can have at
most one message outstanding at a time due to strict alternation, so HWM is
never reached from a single REQ socket.

---

## Scenario 2: REQ — Recv — TCP

`Recv` blocks until the REP server sends a reply. There is no built-in timeout.

**If the REP server is gone:** `Recv` blocks forever. ZMQ reconnects in the
background, but the original request is **not automatically retransmitted** after
reconnect — it was consumed from the REQ buffer at `Send` time. The client
remains blocked until `ZMQ_RCVTIMEO` is configured or the process is killed.

**If HWM (`ZMQ_RCVHWM`) is reached:** ZMQ drops incoming messages for this
socket. For REQ/REP this is uncommon — strict alternation means the REQ receive
queue holds at most one message at a time.

---

## Scenario 3: REQ — Send — IPC

Behaviour is **identical to Scenario 1** (REQ — Send — TCP) in all respects:
lazy connect, buffering, silent drop on disconnect, HWM blocking.

Differences from TCP:
- Endpoint is a filesystem path (`ipc:///tmp/myapp.sock`) instead of host/port.
- Same-machine only.
- Lower latency — IP/TCP stack is bypassed (Unix domain socket).
- Disconnect detection uses Unix socket events instead of TCP FIN/RST, but ZMQ
  handles it identically: silent drop, then reconnect attempt.

---

## Scenario 4: REQ — Recv — IPC

Behaviour is **identical to Scenario 2** (REQ — Recv — TCP): `Recv` blocks
indefinitely if no reply arrives, with the same reconnect and HWM characteristics.

The same-machine constraint and lower latency from Scenario 3 apply here as well.

---

## Scenario 5: REP — Recv — TCP

The REP socket enforces the mirror state machine: `Recv` → `Send` → `Recv` → ...
Calling `Recv` twice in a row returns `EFSM` immediately.

`Recv` blocks until a REQ client sends a request. There is no built-in timeout.
If multiple REQ clients are connected, REP processes them one at a time in
fair-queued order — it will not accept a second request until it has replied to
the first.

**If no client is connected or all clients have gone away:** `Recv` blocks
forever. ZMQ does not return an error when peers disconnect. Configure
`ZMQ_RCVTIMEO` to bound the wait.

**If HWM (`ZMQ_RCVHWM`) is reached:** ZMQ drops incoming messages from clients
whose requests would exceed the limit. The sending REQ client receives no error —
its message is silently discarded.

---

## Scenario 6: REP — Send — TCP

`Send` is called after `Recv` to deliver the reply to the client that sent the
corresponding request. `Send` returns immediately and enqueues the reply in
ZMQ's internal send buffer.

**If the REQ client disconnected while the server was processing the request:**
`Send` still succeeds and returns no error. ZMQ queues the reply briefly, detects
the TCP disconnect, and silently drops the message. This is **not a memory leak**:
the allocation is temporary and bounded by `ZMQ_SNDHWM`. The REP state machine
advances normally to waiting for the next request.

**If HWM (`ZMQ_SNDHWM`) is reached:** `Send` blocks (blocking mode) or returns
`EAGAIN` (non-blocking). In REP/REQ strict alternation, the reply queue holds at
most one message per connected peer, so HWM is rarely reached.

---

## Scenario 7: REP — Recv — IPC

Behaviour is **identical to Scenario 5** (REP — Recv — TCP): blocks until a
client request arrives, same fair-queuing for multiple clients, same HWM drop
semantics.

The same-machine constraint and lower latency from Scenario 3 apply here as well.

---

## Scenario 8: REP — Send — IPC

Behaviour is **identical to Scenario 6** (REP — Send — TCP): `Send` succeeds
even if the client is gone, reply is silently dropped on disconnect, bounded by
HWM.

The same-machine constraint and lower latency from Scenario 3 apply here as well.
