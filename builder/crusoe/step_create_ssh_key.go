package crusoe

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
	"runtime"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"golang.org/x/crypto/ssh"
)

var (
	rsaBits        int         = 2048
	fileMode       int         = 0600
	sshKeyFileMode fs.FileMode = os.FileMode(fileMode)
)

type stepCreateSSHKey struct {
	Debug        bool
	DebugKeyPath string
}

func (s *stepCreateSSHKey) Run(_ context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	config := state.Get("config").(*Config)

	if !config.createTempSSHPair {
		return multistep.ActionContinue
	}

	ui.Say("Generating ephemeral SSH key pair...")

	priv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		errOut := fmt.Errorf("generating ephemeral SSH key: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	privBlk := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(priv),
	}

	pub, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		errOut := fmt.Errorf("generating ephemeral SSH key: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}
	config.Comm.SSHPrivateKey = pem.EncodeToMemory(&privBlk)
	config.Comm.SSHPublicKey = ssh.MarshalAuthorizedKey(pub)

	state.Put("ephemeral_ssh_key_pair", true)
	state.Put("ephemeral_ssh_public_key", string(config.Comm.SSHPublicKey))

	if s.Debug {
		ui.Say(fmt.Sprintf("saving key for debug purposes: %s", s.DebugKeyPath))
		f, err := os.Create(s.DebugKeyPath)
		if err != nil {
			state.Put("error", fmt.Errorf("saving debug key: %w", err))
			return multistep.ActionHalt
		}
		defer f.Close()

		err = pem.Encode(f, &privBlk)
		if err != nil {
			state.Put("error", fmt.Errorf("saving debug key: %w", err))
			return multistep.ActionHalt
		}

		if runtime.GOOS != "windows" {
			if err := f.Chmod(sshKeyFileMode); err != nil {
				state.Put("error", fmt.Errorf("setting permissions of debug key: %w", err))
				return multistep.ActionHalt
			}
		}
	}
	return multistep.ActionContinue
}

func (s *stepCreateSSHKey) Cleanup(state multistep.StateBag) {
	// Key is stored in-memory
}
