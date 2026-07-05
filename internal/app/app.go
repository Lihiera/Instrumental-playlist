package app

import "fmt"

const Name = "instrumental-playlist"

func Run(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("%s: command %q is not implemented yet", Name, args[0])
	}

	fmt.Printf("%s: CLI foundation initialized\n", Name)
	return nil
}
