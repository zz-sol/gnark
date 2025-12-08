// Package main generates a Groth16 proof for the Fib5 circuit and emits a
// Solana-ready Rust verifier (arkworks) plus an embedded test vector under build/.
package main

import (
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fp"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark/backend/groth16"
	groth16bn254 "github.com/consensys/gnark/backend/groth16/bn254"
	"github.com/consensys/gnark/backend/solana"
	bncs "github.com/consensys/gnark/constraint/bn254"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"

	"solana-example/circuits"
)

func main() {
	outDir := filepath.Join("build")
	srcDir := filepath.Join(outDir, "src")
	_ = os.RemoveAll(outDir)
	must(os.MkdirAll(srcDir, 0o755))

	// 1) Compile circuit
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, new(circuits.Fib5))
	must(err)

	r1csTyped, ok := ccs.(*bncs.R1CS)
	if !ok {
		log.Fatalf("unexpected R1CS type %T", ccs)
	}

	// 2) Setup and prove
	pkGeneric, vkGeneric, err := groth16.Setup(r1csTyped)
	must(err)
	pk := pkGeneric.(*groth16bn254.ProvingKey)
	vk := vkGeneric.(*groth16bn254.VerifyingKey)

	assignment := &circuits.Fib5{Out: 5}
	fullWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	must(err)
	proofGeneric, err := groth16.Prove(r1csTyped, pk, fullWitness)
	must(err)
	proof := proofGeneric.(*groth16bn254.Proof)

	publicWitness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	must(err)
	pubVec := publicWitness.Vector().(fr.Vector)

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

	libPath := filepath.Join(srcDir, "lib.rs")
	libFile, err := os.Create(libPath)
	must(err)
	must(vk.ExportSolana(libFile, solana.WithProgramName("fib_verifier"), solana.WithTestVector(tv)))
	must(libFile.Close())

	must(os.WriteFile(filepath.Join(outDir, "Cargo.toml"), []byte(cargoToml), 0o644))

	log.Printf("Generated verifier and test vector in %s.\n", outDir)
	log.Println("Run `cargo test --release` inside that directory to verify the proof with the Rust verifier.")
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
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

const cargoToml = `[package]
name = "fib_verifier"
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
