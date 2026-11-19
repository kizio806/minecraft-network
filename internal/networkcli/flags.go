package networkcli

import (
	"errors"
	"strconv"
)

func parseGlobal(args []string) (project, output string, yes, dry, follow, monitoring bool, lines int, rest []string, err error) {
	lines = 100
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project-dir":
			if i+1 >= len(args) {
				return "", "", false, false, false, false, 0, nil, errors.New("--project-dir requires a value")
			}
			i++
			project = args[i]
		case "--output":
			if i+1 >= len(args) {
				return "", "", false, false, false, false, 0, nil, errors.New("--output requires a value")
			}
			i++
			output = args[i]
		case "--yes":
			yes = true
		case "--dry-run":
			dry = true
		case "--follow":
			follow = true
		case "--monitoring":
			monitoring = true
		case "--lines":
			if i+1 >= len(args) {
				return "", "", false, false, false, false, 0, nil, errors.New("--lines requires a value")
			}
			i++
			lines, err = strconv.Atoi(args[i])
			if err != nil || lines < 0 {
				return "", "", false, false, false, false, 0, nil, errors.New("--lines must be a non-negative integer")
			}
		case "--version":
			rest = append(rest, args[i:]...)
			return
		default:
			rest = append(rest, args[i:]...)
			return
		}
	}
	return
}

func stripFlag(args []string, flag string) ([]string, bool) {
	found := false
	result := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == flag {
			found = true
		} else {
			result = append(result, arg)
		}
	}
	return result, found
}
