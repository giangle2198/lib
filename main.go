package main

import (
	_ "lib/learning/concurrency_patterns"
	_ "lib/learning/race_condition"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixMilli())
}

func main() {
	// rc.NewAtomicFunc(2)
	// rc.NewMutexFunc(2)
	// rc.NewUnbufferedChannel()
	// rc.NewBufferedChannel(4, 10)
	// cp.ProcessRunner(5 * time.Second)
	// cp.ProcessPool(10, 3)
	// cp.ProcessWorkerPool(2)

	// run config vault
	// parameters := config.VaultParameters{
	// 	Address:         "http://127.0.0.1:8200",
	// 	TokenKey:        "hvs.PvriXRGknSy1cH9eQKbVqizg",
	// 	ApiKeyPath:      "bim",
	// 	ApiKeyMountPath: "services",
	// }

	// v, err := config.NewVaultAppRoleClient(context.Background(), parameters)
	// if err != nil {
	// 	panic(err)
	// }

	// data, err := v.GetSecretAPIKeys(context.Background())
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println(data)
}
