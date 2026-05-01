# API Overview

These repo auth config details describe how the service behaves in current env.

## Request Flow

- If the app receives a req without auth, the impl should return a resp with a helpful msg.
- Because the docs is written for external users, the config examples stay concise.
- Now, repo env config examples are available in same pkg for local docs review.
- repo pkg exposes same resp structure for each op.

`POST /v1/login`

```json
{"environment":"prod"}
```
