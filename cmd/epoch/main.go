package main

import (
	"errors"
	"fmt"
	"github.com/SmartPool/smartpool-client"
	"github.com/SmartPool/smartpool-client/ethereum"
	"github.com/SmartPool/smartpool-client/ethereum/geth"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"os"
	"syscall"
)

type Input struct {
	RpcEndPoint  string
	KeystorePath string
	ContractAddr string
	MinerAddr    string
	From         uint
	To           uint
}

func Initialize(c *cli.Context) *Input {
	rpcEndPoint := c.String("rpc")
	keystorePath := c.String("keystore")
	contractAddr := c.String("ethash-contract")
	minerAddr := c.String("account")
	from := c.Uint("from")
	to := c.Uint("to")
	return &Input{
		rpcEndPoint, keystorePath, contractAddr,
		minerAddr, from, to,
	}
}

func promptUserPassPhrase(acc string) (string, error) {
	fmt.Printf("Using account address: %s\n", acc)
	fmt.Printf("Please enter passphrase: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Printf("\n")
	if err != nil {
		return "", err
	} else {
		return string(bytePassword), nil
	}
}

func Run(c *cli.Context) error {
	input := Initialize(c)
	if input.KeystorePath == "" {
		fmt.Printf("You have to specify keystore path by --keystore. Abort!\n")
		return nil
	}
	gasprice := c.Uint("gasprice")
	smartpool.Output = &smartpool.StdOut{}
	address, ok, addresses := geth.GetAddress(
		input.KeystorePath,
		common.HexToAddress(input.MinerAddr),
	)
	if len(addresses) == 0 {
		fmt.Printf("We couldn't find any private keys in your keystore path.\n")
		fmt.Printf("Please make sure your keystore path exists.\nAbort!\n")
		return nil
	}
	fmt.Printf("Using miner address: %s\n", address.Hex())
	input.MinerAddr = address.Hex()
	gethRPC, _ := geth.NewGethRPC(
		input.RpcEndPoint, input.ContractAddr,
		"", big.NewInt(1), "",
	)
	client, err := gethRPC.ClientVersion()
	if err != nil {
		fmt.Printf("Node RPC server is unavailable.\n")
		fmt.Printf("Make sure you have Parity or Geth running.\n")
		return err
	}
	fmt.Printf("Connected to Ethereum node: %s\n", client)
	fmt.Printf("Epoch data will be submitted to contract at %s\n", input.ContractAddr)
	if common.HexToAddress(input.ContractAddr).Big().Cmp(common.Big0) == 0 {
		fmt.Printf("Contract address is not set on gateway. Abort!\n")
		return errors.New("Contract address is not set")
	}
	var ethashContractClient *geth.EthashContractClient
	for {
		if ok {
			passphrase, _ := promptUserPassPhrase(
				input.MinerAddr,
			)
			ethashContractClient, err = geth.NewEthashContractClient(
				common.HexToAddress(input.ContractAddr), gethRPC,
				common.HexToAddress(input.MinerAddr),
				input.RpcEndPoint, input.KeystorePath, passphrase,
				uint64(gasprice),
			)
			if ethashContractClient != nil {
				break
			} else {
				fmt.Printf("error: %s\n", err)
			}
		} else {
			if input.KeystorePath == "" {
				fmt.Printf("You have to specify keystore path by --keystore. Abort!\n")
			} else {
				fmt.Printf("Your keystore: %s\n", input.KeystorePath)
				fmt.Printf("Your miner address: %s\n", input.MinerAddr)
				if len(addresses) > 0 {
					fmt.Printf("We couldn't find the private key of your miner address in the keystore path you specified. We found following addresses:\n")
					for i, addr := range addresses {
						fmt.Printf("%d. %s\n", i+1, addr.Hex())
					}
					fmt.Printf("Please make sure you entered correct miner address.\n")
				} else {
					fmt.Printf("We couldn't find any private keys in your keystore path.\n")
					fmt.Printf("Please make sure your keystore path exists.\nAbort!\n")
				}
			}
			return nil
		}
	}
	ethereumContract := ethereum.NewEthashContract(ethashContractClient)
	for i := int(input.From); i <= int(input.To); i++ {
		fmt.Printf("Calculating epoch datas for epochs number %d...\n", i)
		err = ethereumContract.SetEpochData(i)
		if err != nil {
			fmt.Printf("Got error: %s\n", err)
		} else {
			fmt.Printf("Succeeded.\n")
		}
	}
	return nil
}

func BuildAppCommandLine() *cli.App {
	app := cli.NewApp()
	app.Description = "Commandline tool to calculate and submit epoch data for SmartPool"
	app.Name = "SmartPool epoch tool"
	app.Usage = "Submit epoch data to contract"
	app.Version = smartpool.VERSION
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "rpc",
			Value: "http://localhost:8545",
			Usage: "RPC endpoint of Ethereum node",
		},
		cli.StringFlag{
			Name:  "keystore",
			Usage: "Keystore path to your ethereum account private key. SmartPool will look for private key of the miner address you specified in that path.",
		},
		cli.StringFlag{
			Name:  "account",
			Usage: "The address that is used to submit epoch data (Default: First account in your keystore.)",
		},
		cli.StringFlag{
			Name:  "ethash-contract",
			Value: "0xd93c6293a076fc6c8f49d678b6d01aab3232fddc",
			Usage: "Ethash contract address. Its default value is the official ethash contract maintained by SmartPool team",
		},
		cli.UintFlag{
			Name:  "gasprice",
			Value: 50,
			Usage: "Gas price in gwei to use in communication with the contract. Specify 0 if you let your Ethereum Client decide on gas price.",
		},
		cli.UintFlag{
			Name:  "from",
			Usage: "Starting epoch number to calculate epoch data on.",
		},
		cli.UintFlag{
			Name:  "to",
			Usage: "Ending epoch number to calculate epoch data on.",
		},
	}
	app.Action = Run
	return app
}

func main() {
	app := BuildAppCommandLine()
	app.Run(os.Args)
}
