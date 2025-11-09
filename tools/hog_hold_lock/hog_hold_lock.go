package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/groundhogchain/chuckd/tools/libtools"
)

func main() {
	holder := flag.String("holder", "", "unique name of the lock holder (e.g. random or ci workflow id)")
	hostsStr := flag.String("hosts", "", "comma-separated list of hosts (e.g. hog6,hog7)")
	flag.Parse()

	if *holder == "" || *hostsStr == "" {
		fmt.Fprintln(os.Stderr, "Usage: hog_lock --holder=name --hosts=host1,host2,host3")
		return
	}

	hosts := strings.Split(*hostsStr, ",")
	for i := range hosts {
		hosts[i] = strings.TrimSpace(hosts[i])
	}

	fmt.Fprintf(os.Stdout, "acquiring locks for %s on %v\n", *holder, hosts)
	ctx := context.Background()
	err := libtools.SyncHogLock.HoldLockMany(ctx, *holder, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hog_hold_lock: error holding locks for %s on %v: %v\n", *holder, hosts, err)
		os.Exit(1)
	}
}
