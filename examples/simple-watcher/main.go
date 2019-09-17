package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/replit/go-watchman-client"
)

var (
	DIR  = flag.String("d", "", "directory to watch")
	LIST = flag.Bool("l", false, "list watches instead of starting a watch")
)

func init() {
	flag.Parse()
}

type byTypeAndName []watchman.File

func (x byTypeAndName) Len() int      { return len(x) }
func (x byTypeAndName) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x byTypeAndName) Less(i, j int) bool {
	a := x[i]
	b := x[j]
	if a.Type < b.Type {
		return true
	} else if b.Type < a.Type {
		return false
	}
	return a.Name < b.Name
}

func connect() *watchman.Client {
	os.Stdout.Write([]byte("Connecting to Watchman... "))
	os.Stdout.Sync()
	c, err := watchman.Connect()
	if err != nil {
		fmt.Println("FAILURE")
		die(err)
	}
	fmt.Println("SUCCESS")

	return c
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func listWatches(c *watchman.Client) {
	if roots, err := c.ListWatches(); err != nil {
		die(err)
	} else {
		fmt.Println("Watches:")
		for _, root := range roots {
			fmt.Println("  ", root)
		}
	}
}

func resolveDir() string {
	dir := *DIR
	if dir == "" {
		if wd, err := os.Getwd(); err != nil {
			die(err)
		} else {
			dir = wd
		}
	}

	dir, err := filepath.Abs(dir)
	if err != nil {
		die(err)
	}

	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		die(err)
	}

	return dir
}

func main() {
	c := connect()
	fmt.Printf("version: %s\n\n", c.Version())

	if *LIST {
		listWatches(c)
		return
	}

	dir := resolveDir()
	fmt.Printf("Watching: %s\n\n", dir)
	watch, err := c.AddWatch(dir)
	if err != nil {
		die(err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := range c.Notifications() {
			cn, ok := n.(*watchman.ChangeNotification)
			if !ok || cn.IsFreshInstance {
				continue
			}
			fmt.Printf(
				"Update: (clock=%q)\n",
				cn.Clock,
			)
			files := cn.Files
			sort.Sort(byTypeAndName(files))
			for _, file := range files {
				switch file.Type {
				case "d":
					fmt.Printf("  %9s  %s/\n",
						file.Change, file.Name,
					)
				case "l":
					fmt.Printf("  %9s  %s -> %s\n",
						file.Change, file.Name, file.Target,
					)
				default:
					fmt.Printf("  %9s  %s\n",
						file.Change, file.Name,
					)
				}
			}
			fmt.Println()
		}
	}()

	if _, err = watch.Subscribe("example", dir); err != nil {
		die(err)
	}

	wg.Wait()
}
