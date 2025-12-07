# Solana Groth16 Verifier Example

This standalone example compiles a tiny Fibonacci circuit with gnark, produces a Groth16 proof, and exports a Rust/arkworks verifier with an embedded test vector you can build on Solana. Everything is generated into the `build/` folder so you can inspect the emitted `lib.rs` and run the embedded Rust test.

## Prereqs
- Go 1.21+
- Rust with `cargo` (needed to run the Rust verifier check)

## Generate the verifier + test vector
```bash
cd solana-example
go run ./cmd/generate
```
This writes `build/src/lib.rs` and `build/Cargo.toml` with an embedded proof/public input for `Fib(5)=5`. The generator also prints where the files were written.

## Verify with Rust (off-chain)
```bash
cd build
cargo test --quiet
```
The test uses the embedded vector to confirm the verifier accepts the proof.

If you are offline or behind a restrictive network, `cargo` will fail to fetch dependencies; re-run the Rust step on a machine with crates.io access or use a local crates mirror.

## Using on Solana
Wrap the generated `lib.rs` in a Solana program entrypoint that deserializes proof/public inputs and calls `verify(..)`, then build with your Solana toolchain (`cargo build-sbf`/`cargo-build-bpf`) and deploy. The generated Rust module already exposes `verify`, `verifying_key`, and constants for public input sizing.
