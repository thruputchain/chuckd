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
	holder := flag.String("holder", "", "name of the lock holder (e.g. ci workflow id)")
	hostsStr := flag.String("hosts", "", "comma-separated list of hosts (e.g. hog6,hog7)")
	flag.Parse()

	if *holder == "" || *hostsStr == "" {
		fmt.Fprintln(os.Stderr, "Usage: hog_unlock --holder=name --hosts=host1,host2,host3")
		return
	}

	hosts := strings.Split(*hostsStr, ",")
	for i := range hosts {
		hosts[i] = strings.TrimSpace(hosts[i])
	}

	fmt.Fprintf(os.Stdout, "releasing locks for %s on %v\n", *holder, hosts)
	ctx := context.Background()
	err := libtools.HogUnlockMany(ctx, *holder, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to release locks for %s on %v: %v\n", *holder, hosts, err)
		os.Exit(1)
	}
}
