# No Config Platform — User Stories

## 1. App team adds a new event channel

The checkout team needs a cancellations channel. They edit `no-config-platform.yaml`, adding a new channel entry. `cub-gen` detects the change, traces the new field back to the DRY source, and produces inverse-edit guidance: "to change cancellation channel config in production, edit `no-config-platform-prod.yaml` channels section."

## 2. Production channel rename

Ops renames the EU inbound channel in `no-config-platform-prod.yaml`. The import shows this as an overlay edit with clear lineage: base channel name from `no-config-platform.yaml`, overridden by prod-specific value. The change bundle captures both the before and after state.

## 3. Future: Platform adds channel naming policy

The platform team introduces `platform/policies/channel-naming.yaml` requiring all channels to follow `<service>.<direction>.<region>` naming. On next import, cub-gen reports the policy exists. When connected to ConfigHub, the decision engine can enforce naming compliance before changes reach the provider.

## 4. Credential rotation audit

Security asks: "which services reference `PROVIDER_API_KEY`?" The provenance chain in ConfigHub answers this instantly — the field-origin map traces `credentials.api_key_ref` back to the DRY source, and the governed decision history shows every change to credential references.
