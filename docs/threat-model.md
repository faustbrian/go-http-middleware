# Threat model and resource budgets

Attackers can control request methods, targets, transport headers, bodies,
origins, forwarding fields, encoding preferences, media ranges, timing, and
disconnect behavior. Trusted proxy operators and constructor callers are
privileged but may misconfigure policy.

| Resource | Default or hard bound |
|---|---|
| chain depth | 256 |
| descriptor name | 128 bytes |
| request identifier | 128 bytes default, 1024 hard maximum |
| proxy hops | 16 default, 128 hard maximum |
| parsed policy header | 8192 bytes default, 1 MiB hard maximum |
| CORS values | 64 default, 256 hard maximum |
| compression buffer | 1 MiB default, 16 MiB hard maximum |
| timeout response buffer | caller required, 16 MiB hard maximum |
| route metadata | 128 bytes |
| client class | 64 bytes |
| recovery stack | 64 KiB default, 1 MiB hard maximum |
| in-flight permits/waiters | caller configured, 1,000,000 hard maximum |

Mitigations cover CRLF splitting, spoofed forwarding fields, host confusion,
CORS cache poisoning, middleware-order bypass, panic disclosure, unbounded
draining, compression memory retention, wait queues, and observer cardinality.
Ingress slowloris protection, HTTP request smuggling prevention, TLS, and total
header size remain server/proxy responsibilities.
