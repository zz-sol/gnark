package groth16

import (
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fp"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark/backend/solana"
	"github.com/consensys/gnark/backend/witness"
	cs "github.com/consensys/gnark/constraint/bn254"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// mulCircuit is a tiny circuit used to generate a proof for the Solana export test.
type mulCircuit struct {
	A, B, C frontend.Variable `gnark:",public"`
}

func (c *mulCircuit) Define(api frontend.API) error {
	api.AssertIsEqual(api.Mul(c.A, c.B), c.C)
	return nil
}

func TestExportSolanaWithEmbeddedTestVector(t *testing.T) {
	// This is an integration-style check that runs the BN254 exporter to
	// produce a Rust verifier, then (optionally) calls cargo test to ensure
	// the embedded vector is accepted by arkworks.
	if os.Getenv("GNARK_SKIP_SOLANA_CARGO") == "1" {
		t.Skip("GNARK_SKIP_SOLANA_CARGO set")
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not available, skipping Solana export compilation test")
	}

	assignment := &mulCircuit{A: 2, B: 3, C: 6}

	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, new(mulCircuit))
	if err != nil {
		t.Fatalf("compile circuit: %v", err)
	}
	casted, ok := ccs.(*cs.R1CS)
	if !ok {
		t.Fatalf("unexpected BN254 R1CS type %T", ccs)
	}

	pk := new(ProvingKey)
	vk := new(VerifyingKey)
	if err = Setup(casted, pk, vk); err != nil {
		t.Fatalf("setup: %v", err)
	}

	fullWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		t.Fatalf("full witness: %v", err)
	}
	proof, err := Prove(casted, pk, fullWitness)
	if err != nil {
		t.Fatalf("prove: %v", err)
	}

	publicWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		t.Fatalf("public witness: %v", err)
	}
	pubVec := publicWitness.(witness.Witness).Vector().(fr.Vector)

	tv := solana.TestVector{
		PublicInputs: frSliceToStrings(pubVec),
		Proof: solana.ProofStrings{
			A: [2]string{fpToString(proof.Ar.X), fpToString(proof.Ar.Y)},
			B: [2][2]string{
				{fpToString(proof.Bs.X.A0), fpToString(proof.Bs.X.A1)},
				{fpToString(proof.Bs.Y.A0), fpToString(proof.Bs.Y.A1)},
			},
			C: [2]string{fpToString(proof.Krs.X), fpToString(proof.Krs.Y)},
		},
	}

	tmpDir, err := os.MkdirTemp("", "gnark-solanaverifier-*")
	if err != nil {
		t.Fatalf("create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcDir := filepath.Join(tmpDir, "src")
	if err = os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("create src dir: %v", err)
	}

	libPath := filepath.Join(srcDir, "lib.rs")
	libFile, err := os.Create(libPath)
	if err != nil {
		t.Fatalf("create lib.rs: %v", err)
	}

	if err = vk.ExportSolana(libFile, solana.WithProgramName("gnark_solana_verifier"), solana.WithTestVector(tv)); err != nil {
		t.Fatalf("export solana verifier: %v", err)
	}
	if err = libFile.Close(); err != nil {
		t.Fatalf("close lib.rs: %v", err)
	}

	if err = os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoTomlContent), 0o644); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}

	cmd := exec.Command("cargo", "test", "--quiet")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cargo test failed: %v\n%s", err, string(out))
	}
}

func fpToString(x fp.Element) string {
	bi := new(big.Int)
	x.BigInt(bi)
	return bi.String()
}

func frSliceToStrings(v fr.Vector) []string {
	res := make([]string, len(v))
	for i := range v {
		bi := new(big.Int)
		v[i].BigInt(bi)
		res[i] = bi.String()
	}
	return res
}

const cargoTomlContent = `[package]
name = "gnark_solana_verifier"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["lib"]

[dependencies]
ark-std = "0.5"
ark-ff = "0.5"
ark-ec = "0.5"
ark-serialize = "0.5"
ark-bn254 = "0.5"
ark-groth16 = "0.5"
`
