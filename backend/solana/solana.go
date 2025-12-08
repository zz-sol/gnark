package solana

import "io"

// ExportOption defines options that tune how the Solana verifier is generated.
type ExportOption func(*ExportConfig) error

// ExportConfig controls the generated Solana verifier source.
type ExportConfig struct {
	// ProgramName is used for the generated crate / module name.
	ProgramName string
	// TestVector, when provided, embeds a Rust unit test that verifies the
	// provided proof against the verifying key.
	TestVector *TestVector
}

// TestVector holds values that can be embedded into the generated Solana
// verifier to create a self-checking Rust test.
type TestVector struct {
	// PublicInputs are the public inputs to the circuit encoded as decimal
	// strings in the Fr field.
	PublicInputs []string
	// Proof holds the Groth16 proof elements as decimal strings in Fq.
	Proof ProofStrings
}

// ProofStrings is the string representation of a Groth16 proof.
type ProofStrings struct {
	A [2]string    // G1 point (x, y)
	B [2][2]string // G2 point (x0, x1), (y0, y1)
	C [2]string    // G1 point (x, y)
}

// NewExportConfig builds an ExportConfig applying the provided options.
func NewExportConfig(opts ...ExportOption) (ExportConfig, error) {
	cfg := ExportConfig{
		ProgramName: "gnark_solana_verifier",
	}
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return ExportConfig{}, err
		}
	}
	return cfg, nil
}

// WithProgramName overrides the default crate / module name.
func WithProgramName(name string) ExportOption {
	return func(cfg *ExportConfig) error {
		if name != "" {
			cfg.ProgramName = name
		}
		return nil
	}
}

// WithTestVector embeds a unit test using the provided vector in the generated
// verifier.
func WithTestVector(tv TestVector) ExportOption {
	return func(cfg *ExportConfig) error {
		cfg.TestVector = &tv
		return nil
	}
}

// VerifyingKey mirrors the Solidity package interface to allow generic usage.
type VerifyingKey interface {
	NbPublicWitness() int
	ExportSolana(w io.Writer, opts ...ExportOption) error
}
