package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const svcName = "surgalt"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			installService()
			return
		case "uninstall":
			uninstallService()
			return
		}
	}

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatal(err)
	}
	if isInteractive {
		runApp()
		return
	}
	svc.Run(svcName, &winService{})
}

type winService struct{}

func (ws *winService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}

	exePath, _ := os.Executable()
	dir := filepath.Dir(exePath)
	os.Chdir(dir)

	appPath := filepath.Join(dir, "surgalt.exe")
	cmd := exec.Command(appPath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Println("Failed to start:", err)
		return false, 1
	}

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending}
				cmd.Process.Kill()
				return false, 0
			}
		case <-time.After(time.Second):
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				return false, 1
			}
		}
	}
}

func runApp() {
	exePath, _ := os.Executable()
	dir := filepath.Dir(exePath)
	os.Chdir(dir)

	appPath := filepath.Join(dir, "surgalt.exe")
	cmd := exec.Command(appPath)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func installService() {
	exePath, _ := os.Executable()
	m, err := mgr.Connect()
	if err != nil {
		log.Fatal("mgr.Connect:", err)
	}
	defer m.Disconnect()

	s, err := m.CreateService(svcName, exePath, mgr.Config{
		DisplayName: "Surgalt.mn Platform",
		Description: "Surgalt.mn learning platform Go backend",
		StartType:   mgr.StartAutomatic,
	})
	if err != nil {
		log.Fatal("CreateService:", err)
	}
	defer s.Close()
	log.Println("Service installed!")
}

func uninstallService() {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(svcName)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()
	s.Delete()
	log.Println("Service uninstalled!")
}
