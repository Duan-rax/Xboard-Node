# Managed egress

The managed egress feature binds an administrator-maintained proxy endpoint to
an Xboard node. It is not a user-facing plan or node-sharing feature.

## Behavior

- One profile may be bound to many nodes; one node has at most one profile.
- Credentials are exposed only in the authenticated node control-plane
  response. Subscriptions and user APIs never receive them.
- Private-network blocks, custom routes and panel routes are evaluated before
  the managed egress catch-all rule.
- Unbinding or disabling a profile removes the selected outbound and restores
  the configured final outbound, normally `direct`.
- A node reports the applied profile ID and application timestamp back to the
  panel.
- If a WebSocket update is missed, periodic REST reconciliation fetches the
  current node configuration.

## Hot reload

When an egress changes, the node:

1. Creates or replaces the `panel-egress-*` outbound.
2. Applies the new routing rules.
3. Removes an outbound associated with a previous profile.
4. Keeps the inbound listener and connection tracker running.

The operation does not restart `xboard-node` or its embedded kernel. Existing
client sessions remain connected; connections already using a removed egress
may finish or reconnect through the new route.

## Compatibility

- Use the coordinated Xboard panel release that includes the Egress Management
  page and `/server/egress` APIs.
- Nodes without managed-egress support safely ignore the `egress_proxy` field,
  but binding to them has no effect.
- Do not place egress credentials in `custom_config`. The panel stores them
  encrypted and exposes them only through the node control plane.
- Route blocks, direct overrides and custom rules belong in Xboard's existing
  route management or structured custom-route editor, not in egress profiles.

## Verification

1. Bind one canary node and wait for both the panel acknowledgement and a real
   proxy exit-IP test.
2. Confirm the process PID remains unchanged in node logs.
3. Confirm audit monitoring shows no WebSocket disconnect.
4. Unbind the canary and verify that its original direct exit address returns.
5. Roll out to remaining nodes with per-machine binary backups.
