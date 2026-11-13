# intoto-bundle-demo

This repo builds a small container image, attests it with [`actions/attest`](https://github.com/actions/attest), and records those attestations in [Kosli](https://www.kosli.com) with `kosli attest in-toto`, reading them from the bundle files the build saved.

The job that records in Kosli has only the bundle files and the image's digest. It has no access to the registry, cannot sign anything, and does not ask GitHub for the attestations. The Kosli CLI verifies each bundle's signature against a pinned signer before it records anything.

## What the pipeline does

The workflow is [`.github/workflows/build.yml`](.github/workflows/build.yml). It runs on every push to `main` and on pull requests from branches in this repo.

The `build` job:

1. Builds the image from [`main.go`](main.go) and pushes it to `ghcr.io/kosli-labs/intoto-bundle-demo`, tagged with the commit SHA.
2. Attests the image's SLSA provenance with `actions/attest-build-provenance`. The attestation names the image by its digest and is signed with Sigstore as this workflow.
3. Generates an SPDX SBOM for the image with `anchore/sbom-action` and attests it with `actions/attest`.
4. Saves the two Sigstore bundles as `provenance.sigstore.json` and `sbom.sigstore.json` in a workflow artifact named `attestation-bundles`.
5. Writes the image digest and a link to each attestation's transparency log entry to the run summary.

The `record` job, titled **Record in Kosli from the bundle files**, then records them in the Kosli org `in-toto-demo`:

1. Downloads the `attestation-bundles` artifact.
2. Installs Kosli CLI v2.56.0.
3. Creates the flow `intoto-bundle-demo` from [`kosli-flow.yml`](kosli-flow.yml), which expects an artifact named `image` with an attestation named `sbom`.
4. Begins a trail named after the commit SHA.
5. Records the provenance bundle as the artifact `image`:

   ```shell
   kosli attest in-toto \
     --fingerprint "${DIGEST#sha256:}" \
     --bundle bundles/provenance.sigstore.json \
     --predicate-type provenance \
     --name image \
     --certificate-identity-regexp '^https://github\.com/kosli-labs/intoto-bundle-demo/\.github/workflows/' \
     --certificate-oidc-issuer https://token.actions.githubusercontent.com
   ```

6. Records the SBOM bundle as `image.sbom`, with the same command reading `bundles/sbom.sigstore.json` with `--predicate-type sbom`.

The commit, repository and build URL that Kosli records for the image come from the provenance statement, so they describe the build even though a different job does the recording.

## What to look at when demoing

Start from a workflow run on the [Actions page](https://github.com/kosli-labs/intoto-bundle-demo/actions).

- **The run summary.** It shows the image digest, the predicate type of each attestation, and a link to each one's Rekor entry. The `attestation-bundles` artifact at the bottom of the page holds the two bundle files.
- **The repo's [Attestations page](https://github.com/kosli-labs/intoto-bundle-demo/attestations).** Each run adds two attestations for the image digest: one SLSA provenance, one SPDX SBOM.
- **The Rekor transparency log entry.** This repo is public, so `actions/attest` signs with Sigstore's public instance and logs each signature in Rekor. Follow a link from the run summary to `search.sigstore.dev`. The entry's certificate names this workflow as the signer.
- **The record in Kosli.** In the org `in-toto-demo`, in the flow `intoto-bundle-demo`, open the trail named after the commit SHA. The artifact `image` carries the image digest as its fingerprint, with the commit and build URL from the provenance. The attestation `sbom` has the bundle file attached. Both are annotated `verified: true` with the `signer_identity`, `signer_issuer` and Rekor `log_index` that the CLI checked, and both link to the Rekor entry as the external URL `attestation`.

For a pull request, the commit SHA is that of the pull request's merge commit, which is what the workflow builds.

## Kosli configuration

The `record` job reads the Kosli org and API token from two settings under **Settings › Secrets and variables › Actions**. Both are set on this repo:

| Kind | Name | Value |
| --- | --- | --- |
| Repository secret | `KOSLI_API_TOKEN` | An API token for the Kosli org |
| Repository variable | `KOSLI_ORG` | The name of the Kosli org: `in-toto-demo` |

In a copy of this repo without them, the `record` job is skipped and the workflow stays green: the image is still built, attested and its bundles saved. Set both to record in your own Kosli org.

## Try a bundle yourself

Download the `attestation-bundles` artifact from a run and copy the image digest from the run summary. With Kosli CLI v2.56.0 or later, `--dry-run` verifies the bundle and prints what would be recorded without sending it:

```shell
kosli attest in-toto \
  --fingerprint <digest-without-sha256-prefix> \
  --bundle provenance.sigstore.json \
  --predicate-type provenance \
  --name image \
  --certificate-identity-regexp '^https://github\.com/kosli-labs/intoto-bundle-demo/\.github/workflows/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --flow intoto-bundle-demo \
  --trail <commit-sha> \
  --org <your-org> \
  --api-token <your-api-token> \
  --dry-run
```

Pass a different signer identity and the command reports that the attestation failed verification instead of printing a payload. Without `--dry-run` it would exit non-zero and record nothing.
