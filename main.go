package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
)

type Deamon struct {
	Location string `json:"location"`
	Name     string `json:"name"`
}

var snapshotDir = fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".launchm")
var snapshotFile = fmt.Sprintf("%s/%s", snapshotDir, "snapshot.json")

func snapshot(deamons []Deamon) error {
	file, err := json.MarshalIndent(deamons, "", " ")
	if err != nil {
		return err
	}

	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		err := os.Mkdir(snapshotDir, 0755)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(snapshotFile, file, 0644)
	if err != nil {
		return err
	}

	return nil
}

func readSnapshot() ([]Deamon, error) {
	var deamons []Deamon

	f, err := os.Open(snapshotFile)
	if err != nil {
		return deamons, err
	}
	defer f.Close()

	bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return deamons, err
	}

	err = json.Unmarshal(bytes, &deamons)
	if err != nil {
		return deamons, err
	}

	return deamons, nil
}

func getAllDemons() ([]Deamon, error) {
	plistPaths := []string{
		path.Join(os.Getenv("HOME"), "Library", "LaunchAgents"),
		path.Join("/System", "Library", "LaunchAgents"),
		path.Join("/System", "Library", "LaunchDaemons"),
		path.Join("/", "Library", "LaunchAgents"),
		path.Join("/", "Library", "LaunchDaemons"),
	}

	var deamons []Deamon

	for _, path := range plistPaths {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return deamons, err
		}
		for _, file := range files {
			deamons = append(deamons, Deamon{
				Name:     file.Name(),
				Location: path,
			})
		}
	}

	return deamons, nil
}

func compare(a, b []Deamon) []Deamon {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x.Name] = struct{}{}
	}

	var diff []Deamon
	for _, x := range a {
		if _, found := mb[x.Name]; !found {
			diff = append(diff, x)
		}
	}

	return diff
}

func main() {
	helpMsg := "all - list all existing services\napshot - Create snapshot of all currently installed deamons.\nclean - Remove all new deamons that differ from the snapshot.\n"

	curDeamons, err := getAllDemons()
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 {
		if _, err := os.Stat(snapshotFile); os.IsNotExist(err) {
			if err := snapshot(curDeamons); err != nil {
				log.Fatal(err)
			}
		}

		snapDeamonst, err := readSnapshot()
		if err != nil {
			log.Fatal(err)
		}

		diff := compare(curDeamons, snapDeamonst)
		for _, d := range diff {
			fmt.Printf("%s/%s\n", d.Location, d.Name)
		}
	} else {

		switch os.Args[1] {

		case "all":
			for _, deamon := range curDeamons {
				fmt.Printf("%s/%s\n", deamon.Location, deamon.Name)
			}

		/*
		 * Make a snapshot of currently installed plists
		 */
		case "snapshot":
			if err := snapshot(curDeamons); err != nil {
				log.Fatal(err)
			}

			snapDeamonst, err := readSnapshot()
			if err != nil {
				log.Fatal(err)
			}

			diff := compare(curDeamons, snapDeamonst)
			for _, d := range diff {
				fmt.Printf("%s/%s\n", d.Location, d.Name)
			}

		/*
		 * Unload and remove plist files
		 */
		case "clean":
			snapDeamonst, err := readSnapshot()
			if err != nil {
				log.Fatal(err)
			}
			diff := compare(curDeamons, snapDeamonst)

			for _, d := range diff {
				service := fmt.Sprintf("%s/%s", d.Location, d.Name)

				cmd := exec.Command("launchctl", "unload", service)

				out, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("[-] can not unload service. ", err)
				}

				if len(out) > 0 {
					fmt.Println(string(out))
				}

				err = os.Remove(service)
				if err != nil {
					fmt.Printf("[-] can not remove plist. ", err)
				}
			}

		default:
			fmt.Println(helpMsg)
			os.Exit(1)
		}
	}

}
