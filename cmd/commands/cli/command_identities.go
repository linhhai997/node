/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package cli

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/pkg/errors"
)

func (c *cliApp) identities(argsString string) {
	var usage = strings.Join([]string{
		"Usage: identities <action> [args]",
		"Available actions:",
		"  " + usageListIdentities,
		"  " + usageGetIdentity,
		"  " + usageNewIdentity,
		"  " + usageUnlockIdentity,
		"  " + usageRegisterIdentity,
		"  " + usageSettle,
		"  " + usageGetReferralCode,
		"  " + usageExportIdentity,
		"  " + usageImportIdentity,
	}, "\n")

	if len(argsString) == 0 {
		clio.Info(usage)
		return
	}

	args := strings.Fields(argsString)
	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "list":
		c.listIdentities(actionArgs)
	case "get":
		c.getIdentity(actionArgs)
	case "new":
		c.newIdentity(actionArgs)
	case "unlock":
		c.unlockIdentity(actionArgs)
	case "register":
		c.registerIdentity(actionArgs)
	case "beneficiary":
		c.setBeneficiary(actionArgs)
	case "settle":
		c.settle(actionArgs)
	case "referralcode":
		c.getReferralCode(actionArgs)
	case "export":
		c.exportIdentity(actionArgs)
	case "import":
		c.importIdentity(actionArgs)
	default:
		clio.Warnf("Unknown sub-command '%s'\n", argsString)
		fmt.Println(usage)
	}
}

const usageListIdentities = "list"

func (c *cliApp) listIdentities(args []string) {
	if len(args) > 0 {
		clio.Info("Usage: " + usageListIdentities)
		return
	}
	ids, err := c.tequilapi.GetIdentities()
	if err != nil {
		fmt.Println("Error occurred:", err)
		return
	}

	for _, id := range ids {
		clio.Status("+", id.Address)
	}
}

const usageGetIdentity = "get <identity>"

func (c *cliApp) getIdentity(actionArgs []string) {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageGetIdentity)
		return
	}

	address := actionArgs[0]
	identityStatus, err := c.tequilapi.Identity(address)
	if err != nil {
		clio.Warn(err)
		return
	}
	clio.Info("Registration Status:", identityStatus.RegistrationStatus)
	clio.Info("Channel address:", identityStatus.ChannelAddress)
	clio.Info(fmt.Sprintf("Balance: %s", money.New(identityStatus.Balance)))
	clio.Info(fmt.Sprintf("Earnings: %s", money.New(identityStatus.Earnings)))
	clio.Info(fmt.Sprintf("Earnings total: %s", money.New(identityStatus.EarningsTotal)))
}

const usageNewIdentity = "new [passphrase]"

func (c *cliApp) newIdentity(args []string) {
	if len(args) > 1 {
		clio.Info("Usage: " + usageNewIdentity)
		return
	}
	passphrase := identityDefaultPassphrase
	if len(args) == 1 {
		passphrase = args[0]
	}

	id, err := c.tequilapi.NewIdentity(passphrase)
	if err != nil {
		clio.Warn(err)
		return
	}
	clio.Success("New identity created:", id.Address)
}

const usageUnlockIdentity = "unlock <identity> [passphrase]"

func (c *cliApp) unlockIdentity(actionArgs []string) {
	if len(actionArgs) < 1 {
		clio.Info("Usage: " + usageUnlockIdentity)
		return
	}

	address := actionArgs[0]
	var passphrase string
	if len(actionArgs) >= 2 {
		passphrase = actionArgs[1]
	}

	clio.Info("Unlocking", address)
	err := c.tequilapi.Unlock(address, passphrase)
	if err != nil {
		clio.Warn(err)
		return
	}

	clio.Success(fmt.Sprintf("Identity %s unlocked.", address))
}

const usageRegisterIdentity = "register <identity> [stake] [beneficiary] [referralcode]"

func (c *cliApp) registerIdentity(actionArgs []string) {
	if len(actionArgs) < 1 || len(actionArgs) > 4 {
		clio.Info("Usage: " + usageRegisterIdentity)
		return
	}

	var address = actionArgs[0]
	stake := new(big.Int).SetInt64(0)
	if len(actionArgs) >= 2 {
		s, ok := new(big.Int).SetString(actionArgs[1], 10)
		if !ok {
			clio.Warn("could not parse stake")
		}
		stake = s
	}
	var beneficiary string
	if len(actionArgs) >= 3 {
		beneficiary = actionArgs[2]
	}

	var token *string
	if len(actionArgs) >= 4 {
		token = &actionArgs[3]
	}

	err := c.tequilapi.RegisterIdentity(address, beneficiary, stake, token)
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not register identity"))
		return
	}

	msg := "Registration started. Topup the identities channel to finish it."
	if c.config.GetBoolByFlag(config.FlagTestnet2) || c.config.GetBoolByFlag(config.FlagTestnet) {
		msg = "Registration successful, try to connect."
	}

	clio.Info(msg)
	clio.Info(fmt.Sprintf("To explore additional information about the identity use: %s", usageGetIdentity))
}

const usageSettle = "settle <providerIdentity>"

func (c *cliApp) settle(args []string) {
	if len(args) != 1 {
		clio.Info("Usage: " + usageSettle)
		fees, err := c.tequilapi.GetTransactorFees()
		if err != nil {
			clio.Warn("could not get transactor fee: ", err)
		}
		trFee := new(big.Float).Quo(new(big.Float).SetInt(fees.Settlement), new(big.Float).SetInt(money.MystSize))
		hermesFee := new(big.Float).Quo(new(big.Float).SetInt(big.NewInt(int64(fees.Hermes))), new(big.Float).SetInt(money.MystSize))
		clio.Info(fmt.Sprintf("Transactor fee: %v MYST", trFee.String()))
		clio.Info(fmt.Sprintf("Hermes fee: %v MYST", hermesFee.String()))
		return
	}
	hermesID, err := c.config.GetHermesID()
	if err != nil {
		clio.Warn("could not get hermes id: ", err)
		return
	}
	clio.Info("Waiting for settlement to complete")
	errChan := make(chan error)

	go func() {
		errChan <- c.tequilapi.Settle(identity.FromAddress(args[0]), identity.FromAddress(hermesID), true)
	}()

	timeout := time.After(time.Minute * 2)
	for {
		select {
		case <-timeout:
			fmt.Println()
			clio.Warn("Settlement timed out")
			return
		case <-time.After(time.Millisecond * 500):
			fmt.Print(".")
		case err := <-errChan:
			fmt.Println()
			if err != nil {
				clio.Warn("settlement failed: ", err.Error())
				return
			}
			clio.Info("settlement succeeded")
			return
		}
	}
}

const usageGetReferralCode = "referralcode <identity>"

func (c *cliApp) getReferralCode(actionArgs []string) {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageGetReferralCode)
		return
	}

	address := actionArgs[0]
	res, err := c.tequilapi.IdentityReferralCode(address)
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not get referral token"))
		return
	}

	clio.Success(fmt.Sprintf("Your referral token is: %q", res.Token))
}

func (c *cliApp) setBeneficiary(actionArgs []string) {
	const usageSetBeneficiary = "beneficiary <identity> <new beneficiary>"

	if len(actionArgs) < 2 || len(actionArgs) > 3 {
		clio.Info("Usage: " + usageSetBeneficiary)
		return
	}

	address := actionArgs[0]
	beneficiary := actionArgs[1]
	hermesID, err := c.config.GetHermesID()
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not get hermes id"))
		return
	}
	err = c.tequilapi.SettleWithBeneficiary(address, beneficiary, hermesID)
	if err != nil {
		clio.Warn(errors.Wrap(err, "could not set beneficiary"))
		return
	}

	clio.Info("Waiting for new beneficiary to be set")
	timeout := time.After(1 * time.Minute)

	for {
		select {
		case <-timeout:
			clio.Warn("Setting new beneficiary timed out")
			return
		case <-time.After(time.Second):
			data, err := c.tequilapi.Beneficiary(address)
			if err != nil {
				clio.Warn(err)
			}

			if strings.EqualFold(data.Beneficiary, beneficiary) {
				clio.Success("New beneficiary address set")
				return
			}

			fmt.Print(".")
		}
	}
}

const usageExportIdentity = "export <identity> <new_passphrase> [file]"

func (c *cliApp) exportIdentity(actionsArgs []string) {
	dataDir := c.config.GetStringByFlag(config.FlagDataDir)
	if dataDir == "" {
		clio.Error("Could not get data directory")
		return
	}

	if len(actionsArgs) < 2 || len(actionsArgs) > 3 {
		clio.Info("Usage: " + usageExportIdentity)
		return
	}
	id := actionsArgs[0]
	passphrase := actionsArgs[1]

	ksdir := node.GetOptionsDirectoryKeystore(dataDir)
	ks := keystore.NewKeyStore(ksdir, keystore.LightScryptN, keystore.LightScryptP)

	ex := identity.NewExporter(identity.NewKeystoreFilesystem(ksdir, ks))

	blob, err := ex.Export(id, "", passphrase)
	if err != nil {
		clio.Error("Failed to export identity: ", err)
		return
	}

	if len(actionsArgs) == 3 {
		filepath := actionsArgs[2]
		write := func() error {
			f, err := os.Create(filepath)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.Write(blob)
			return err
		}

		err := write()
		if err != nil {
			clio.Error(fmt.Sprintf("Failed to write exported key to file: %s reason: %s", filepath, err.Error()))
			return
		}

		clio.Success("Identity exported to file:", filepath)
		return
	}

	clio.Success("Private key exported: ")
	fmt.Println(string(blob))
}

const usageImportIdentity = "import <passphrase> <key-string/key-file>"

func (c *cliApp) importIdentity(actionsArgs []string) {
	if len(actionsArgs) != 2 {
		clio.Info("Usage: " + usageImportIdentity)
		return
	}

	key := actionsArgs[1]
	passphrase := actionsArgs[0]

	blob := []byte(key)
	if _, err := os.Stat(key); err == nil {
		blob, err = ioutil.ReadFile(key)
		if err != nil {
			clio.Error(fmt.Sprintf("Can't read provided file: %s reason: %s", key, err.Error()))
			return
		}
	}

	id, err := c.tequilapi.ImportIdentity(blob, passphrase, true)
	if err != nil {
		clio.Error("Failed to import identity: ", err)
		return
	}

	clio.Success("Identity imported:", id.Address)
}
