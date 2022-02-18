package main

var (
	// version of restic-scheduler being run
	version = "dev"
)

func main() {
	showVersion := flag.Bool("version", false, "Display the version of minitor and exit")
	flag.Parse()

	// Print version if flag is provided
	if *showVersion {
		fmt.Println("restic-scheduler version:", version)

		return
	}
}
