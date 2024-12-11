// Script handles Minecraft server deployment:
// PaperMC, Forge, Fabric.

// flag usage:
// -userID (obviously the user ID)
// -userServerID (The users server ID relating to their hosting)
// -userServerType (Vanilla, PaperMC or Forge - numerical IDs are used (1,2,3))
// -userServerPort (The port of the game server instance)
// -userServerXMS (Minimum amount of RAM allocated)
// -userServerXMX (Maximum amount of RAM allocated)
// -userServerThreads (Dedicated paralleled CPU threads)

// TODO non hardcoded paths
// TODO UUID Logging

package main

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

type SetupTmpl struct {
	XMS string 
	XMX string
	Threads string
	Jar string
	Option string
}

type ServiceTmpl struct {
	UserID string
	UserServerID string
}

type ServerDotProperties struct {
	QueryPort: string
	RconPort: string
	ServerPort: string
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_-+={}[/?]"

func gen(length int) (string, error) {
	product := make([]byte, length)
	for i := range product {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		product[i] = charset[randomIndex.Int64()]
	}
	return string(product), nil
}

func copyFile(src string, dest string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "copyFile")

	bytesRead, err := os.ReadFile(src)
	if err != nil {
		logger.Warn("Error reading source file", err.Error())
		return err
	}

	err = os.WriteFile(dest, bytesRead, 0644)
	if err != nil {
		logger.Warn("Error writing destination file", err.Error())
		return err
	}

	return nil

}

func checkFile(prefix string, suffix string, path string) (string, error) {
	//New logger instance
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "checkFile")
	logger.Info(fmt.Sprintf("Checking for file: %v with extension: %v in location: %v", prefix, suffix, path))

	//setup the request for the jar search
	request := filepath.Join(".", fmt.Sprintf("%v*%v", prefix, suffix))
	jarFile, err := filepath.Glob(request)
	if err != nil {
		logger.Warn("Error running file search request", "error", err.Error())
		return "", err
	}

	//check if the jar file exists within the search results
	if len(jarFile) == 0 {
		err = fmt.Errorf("failed to find jarfile")
		logger.Warn("Could not find jarFile", "path", path)
		return "", err
	}

	foundJar := jarFile[0]
	logger.Info(fmt.Sprintf("Found jar file: %v", foundJar))

	return foundJar, nil
}

func makeStartScript(makeContent string) error {
	//New logger instance
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "MakeScript")
	//Create the file
	file, err := os.Create("start.sh")
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}
	defer file.Close()

	//Write file content
	_, err = file.WriteString(makeContent)
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}

	// Set exec permissions
	err = os.Chmod("start.sh", 0755)
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}

	return nil

}

func isPaper(userServerXMS string, userServerXMX string, path string, userServerThreads string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "isPaper")

	sourceFile := "../resources/paper.jar"
	destination := filepath.Join(path, "paper.jar")
	err := copyFile(sourceFile, destination)
	if err != nil {
		logger.Warn("Error", "Moving paper.jar failed", err.Error())
		return err
	}

	paperTmpl := []SetupTmpl{
		{
			XMS: *userServerXMS,
			XMX: *userServerXMX,
			Threads: *userServerThreads,
			Jar: "paper.jar",
			Option: "",
		},
	}

	var tmplFile = "../resources/templates/setup.tmpl"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		logger.Warn("Error", "Parsing tmpl.file failed", err.Error())
		os.Exit(1)
	}
	filepath = filepath.Join(path, "start.sh")
	file, err := os.Create(filepath)
	if err != nil {
		logger.Warn("Error", "Failed to create start.sh", err.Error())
		os.Exit(1)
	}
	err = tmpl.Execute(file, paperTmpl) 
	if err != nil {
		logger.Warn("Error, ", "Template execution failed", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	logger.Info("start.sh created successfully!")
	return nil

}
func isForge(userServerXMS string, userServerXMX string, path string, userServerThreads string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "isForge")

	sourceFile := "../resources/forgeInstaller.jar"
	destination := filepath.Join(path, "installer.jar")
	err := copyFile(sourceFile, destination)
	if err != nil {
		logger.Warn("Error", "Moving forgeInstaller.jar failed", err.Error())
		return err
	}

	ForgeTmpl := []SetupTmpl{
		{
			XMS: *userServerXMS, 
			XMX: *userServerXMX, 
			Threads: *userServerThreads, 
			Jar: "-installer.jar",
			Option: "--installServer",
		},
	}
	var tmplFile = "../resources/templates/setup.tmpl"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		logger.Warn("Error", "Parsing tmpl.file failed", err.Error())
		os.Exit(1)
	}
	filepath = filepath.Join(path, "start.sh")
	file, err := os.Create(filepath)
	if err != nil {
		logger.Warn("Error", "Failed to create ForgeInstaller.sh", err.Error())
		os.Exit(1)
	}
	err = tmpl.Execute(file, ForgeTmpl) 
	if err != nil {
		logger.Warn("Error, ", "Template execution failed", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	logger.Info("start.sh created successfully!")
	return nil
	cmd := exec.Command("chmod", "0644", fmt.Sprintf("%v", path))
	err = cmd.Run()
	if err != nil {
		logger.Warn("Error setting executable for ForgeInstall: /start.sh")
		os.Exit(1)
		return err
	}

	cmd = exec.Command("sh", "./start.sh")
	err = cmd.Run()
	if err != nil {
		logger.Info("Failed to install Forge Server")
		logger.Warn("Install failed", "error", err.Error())
		os.Exit(1)
		return err
	}

	//check for new Forge{version}.jar from the installer
	foundJar, err := checkFile("forge", ".jar", path)
	if err != nil {
		logger.Warn("Cannot run checkFile", "error", err.Error())
		os.Exit(1)
		return err
	}

	//remove installer.jar start.sh run.bat installer.log.jar
	cmd = exec.Command("rm", "-r", "installer.jar")
	err = cmd.Run()
	if err != nil {
		logger.Warn("Error removing installer.jar - is it still in use?", "error", err.Error())
		os.Exit(1)
	}
	cmd = exec.Command("rm", "-r", "run.bat")
	err = cmd.Run()
	if err != nil {
		logger.Warn("Error removing run.bat", "error", err.Error())
		os.Exit(1)
	}
	cmd = exec.Command("rm", "-r", "installer.jar.log")
	err = cmd.Run()
	if err != nil {
		logger.Warn("Error removing installer.jar.log", "error", err.Error())
		os.Exit(1)
	}
	cmd = exec.Command("rm", "-r", "start.sh")
	err = cmd.Run()
	if err != nil {
		logger.Warn("Error removing start.sh", "error", err.Error())
		os.Exit(1)
	}

	ForgeTmpl := []SetupTmpl{
		{
			XMS: *userServerXMS, 
			XMX: *userServerXMX, 
			Threads: *userServerThreads, 
			Jar: foundJar,
			Option: "",
		},
	}
	var tmplFile = "../resources/templates/setup.tmpl"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		logger.Warn("Error", "Parsing tmpl.file failed", err.Error())
		os.Exit(1)
	}
	filepath = filepath.Join(path, "start.sh")
	file, err := os.Create(filepath)
	if err != nil {
		logger.Warn("Error", "Failed to create ForgeInstaller.sh", err.Error())
		os.Exit(1)
	}
	err = tmpl.Execute(file, ForgeTmpl) 
	if err != nil {
		logger.Warn("Error, ", "Template execution failed", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	logger.Info("start.sh content written successfully!")
	return nil
}

func isFabric(userServerXMS string, userServerXMX string, path string, userServerThreads string) error {
	//init logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "isFabric")
	logger.Info(fmt.Sprintf("isFabric started with args: %v, %v, %v", userServerXMS, userServerXMX, path))

	sourceFile := "../resources/forgeInstaller.jar"
	destination := filepath.Join(path, "installer.jar")
	err := copyFile(sourceFile, destination)
	if err != nil {
		logger.Warn("Error", "Moving forgeInstaller.jar failed", err.Error())
		return err
	}

	FabricTmpl := []SetupTmpl{
		{
			XMS: *userServerXMS, 
			XMX: *userServerXMX, 
			Threads: *userServerThreads, 
			Jar: foundJar,
			Option: "nogui",
		},
	}
	var tmplFile = "../resources/templates/setup.tmpl"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		logger.Warn("Error", "Parsing tmpl.file failed", err.Error())
		os.Exit(1)
	}
	filepath = filepath.Join(path, "start.sh")
	file, err := os.Create(filepath)
	if err != nil {
		logger.Warn("Error", "Failed to create ForgeInstaller.sh", err.Error())
		os.Exit(1)
	}
	err = tmpl.Execute(file, ForgeTmpl) 
	if err != nil {
		logger.Warn("Error, ", "Template execution failed", err.Error())
		os.Exit(1)
	}
	defer file.Close()

}

func makeService(userID string, userServerID string, socketName string) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "Service Maker")

	service := []ServiceTmpl{
		{
			UserID: *userID,
			UserServerID: *userServerID,
		},
	}
	var tmplFile = "../resources/templates/service.tmpl"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		logger.Warn("Error", "Parsing service.tmpl file failed", err.Error())
		os.Exit(1)
	}
	filepath = filepath.Join(path, fmt.Sprintf(*userID,"_",*userServerID,".service"))
	file, err := os.Create(filepath)
	if err != nil {
		logger.Warn("Error", "Failed to create service file", err.Error())
		os.Exit(1)
	}
	err = tmpl.Execute(file, service) 
	if err != nil {
		logger.Warn("Error, ", "service.tmpl execution failed", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	logger.Info("Service file created for user", "%v", userID)

	logger.Info("Setting service owner as", "%v", userID)

	cmd := exec.Command("chown", fmt.Sprintf("%v:%v", userID, userID), serviceFilePath)
	err = cmd.Run()
	if err != nil {
		logger.Warn(err.Error())
		logger.Info("Failed to chown file %v for user %v", serviceFilePath, userID)
		os.Exit(1)
		return err
	}
	logger.Info("Service file chown success with user: ", "%v", userID)

	logger.Info("Changing service file permissions 700")
	cmd = exec.Command("chmod", "700", fmt.Sprintf("%v", serviceFilePath))
	err = cmd.Run()
	if err != nil {
		logger.Warn("Error using chmod for serviceFile", "Error", err.Error())
		os.Exit(1)
	}
	//reload systemctl daemon
	logger.Info("Restarting systemctl daemon..")
	cmd = exec.Command("systemctl", "daemon-reload")
	err = cmd.Run()
	if err != nil {
		logger.Warn("Failed reloading systemctl daemon", "error", err.Error())
		os.Exit(1)
		return err
	}

	//enable systemd service
	logger.Info("trying to enable the new service...")
	cmd = exec.Command("/bin/systemctl", "enable", fmt.Sprintf("%v.service", socketName))
	err = cmd.Run()
	if err != nil {
		logger.Info(fmt.Sprintf("Cannot run enableService for user %v", userID))
		logger.Warn(err.Error())
		os.Exit(1)
	}
	logger.Info("Systemd service enabled..")

	//start systemd service
	logger.Info("starting systemd service")
	cmd = exec.Command("/bin/systemctl", "start", fmt.Sprintf("%v.service", socketName))
	err = cmd.Run()
	if err != nil {
		logger.Info("error starting systemd service", "user", socketName)
		logger.Warn(err.Error())
		os.Exit(1)
	}

	return nil
}

func makeConfig(userServerPort string, path string) error {
	//initialise logger for the function
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	logger = slog.With("logID", "makeConfig")
	logger.Info("creating server.properties")

	//convert port from string to int then +1 then convert back
	rconPort, err := strconv.Atoi(userServerPort)
	if err != nil {
		logger.Warn("Failed to incerement userServerPort", "error", err.Error())
		os.Exit(1)
	}
	rconPort++
	rconString := strconv.Itoa(rconPort)
	rconPort++
	queryString := strconv.Itoa(rconPort)

	config := []ServerDotProperties{
		{
			QueryPort: *queryString,
			RconPort: *rconString, 
			ServerPort: *userServerPort,
		},
	}
	var tmplFile = "../resources/templates/serverProperties.tmpl"
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		logger.Warn("Error", "Parsing serverProperties.tmpl file failed", err.Error())
		os.Exit(1)
	}
	filepath = filepath.Join(path, "server.properties"))
	file, err := os.Create(filepath)
	if err != nil {
		logger.Warn("Error", "Failed to create server.properties file", err.Error())
		os.Exit(1)
	}
	err = tmpl.Execute(file, config) 
	if err != nil {
		logger.Warn("Error, ", "serverProperties.tmpl execution failed", err.Error())
		os.Exit(1)
	}
	defer file.Close()
	logger.Info("server.properties created")

	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// TODO: Actual dynamic log Ids (preferrably UUID)
	slog.SetDefault(logger)
	logger = slog.With("logID", "Main")

	userID := flag.String("userID", "", "User ID")
	userServerID := flag.String("userServerID", "", "User Server ID")
	userServerType := flag.Int("userServerType", 0, "User Server Type 1-3")
	userServerPort := flag.String("userServerPort", " ", "Users game service firewall port")
	userServerXMS := flag.String("userServerXMS", "", "Users game service minimum memory allowance")
	userServerXMX := flag.String("userServerXMX", "", "Users game service maximimum memory allowance")
	userServerThreads := flag.String("userServerThreads", "", "Amount of dedicated paralleled threads")

	flag.Parse()

	logger.Info("parsed flags", "userId", *userID, "userServerId", *userServerID, "userServerType", *userServerType, "userServicePort", *userServerPort, "userServerXMS", *userServerXMS, "userServerXMX", userServerXMX, "userServerThreads", userServerThreads)

	if *userID == "" || *userServerID == "" || *userServerType == 0 || *userServerPort == "" || *userServerXMS == "" || *userServerXMX == "" || *userServerThreads == ""{
		logger.Info("-userID -userServerID -userServerType -userServerPort -userServerXMS -userServerXMX -userServerThreads are required")
		os.Exit(1)
	}

	cmd := exec.Command("ufw", "allow", fmt.Sprintf("%v", *userServerPort))
	err := cmd.Run()
	if err != nil {
		logger.Info("Unable to run ufw allow", "port", *userServerPort)
		logger.Warn(err.Error())
		os.Exit(1)
	}

	// Create users home directory
	path := fmt.Sprintf("/home/servers/%v/%v", *userID, *userServerID)
	err = os.MkdirAll(path, 0755)
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}
	logger.Info("User directory created")

	var newUser *user.User                                                        //CHANGE ME -k skeleton dir
	cmd = exec.Command("useradd", "-d", path, "-s", "/usr/sbin/nologin", *userID) //CHANGE ME createUser(uname, udir string){}

	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb

	err = cmd.Run()
	if err != nil {
		//Log the error output from the command
		logger.Error("user creation failure", "err", errb.String(), "output", outb.String(), "msg", err.Error())
		os.Exit(1)
	} // search for the newly created userID's system UID
	newUser, err = user.Lookup(*userID)
	if err != nil {
		logger.Warn(err.Error())
		logger.Info("Error looking up UID for", "user", userID)
	}
	logger.Info("Success, created:", "user", *userID, "with UID:", newUser.Uid)

	//TODO add input sanitization - print and check that the userID and PATH are correct before running the command
	cmd = exec.Command("usermod", "-d", path, *userID) //TODO add -z for selinux
	err = cmd.Run()
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}
	logger.Info("Usermod success! directory set to %s", path, path)

	cmd = exec.Command("chown", "-R", fmt.Sprintf("%v:%v", *userID, *userID), path)
	err = cmd.Run()
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}
	logger.Info("Chown success!")

	cmd = exec.Command("chmod", "-R", "-t", "0750", path)
	err = cmd.Run()
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}
	logger.Info("Chmod success!")

	err = os.Chdir(path)
	if err != nil {
		logger.Warn(err.Error())
		os.Exit(1)
	}
	logger.Info("Changed directory to user home")

	//TODO change to local file transfer instead of script generated
	logger.Info("creating eula")
	cmd = exec.Command("sh", "-c", "echo 'eula=true' >> eula.txt")
	err = cmd.Run()
	if err != nil {
		logger.Warn(err.Error())
		logger.Info("Failed to create eula.txt")
		os.Exit(1)
	}

	// move server.properties template over
	err = makeConfig(*userServerPort, path)
	if err != nil {
		logger.Warn("Unable to run", "makeConfig", err.Error())
	}

	//check userServerType to see what type of server is being setup
	//isPaper()
	if *userServerType == 1 {
		err = isPaper(*userServerXMS, *userServerXMX, path)
		if err != nil {
			logger.Warn("Unable to run isPaper()", "error", err.Error())
		}

	}

	if *userServerType == 2 {
		err = isForge(*userServerXMS, *userServerXMX, path)
		if err != nil {
			logger.Warn("Unable to run isForge()", "error", err.Error())
			os.Exit(1)
		}
	}

	if *userServerType == 3 {
		err = isFabric(*userServerXMS, *userServerXMX, path)
		if err != nil {
			logger.Warn("Unable to run isFabric()", "error", err.Error())
			os.Exit(1)
		}
	}

	socketName := strings.Join([]string{*userID, *userServerID}, "_")
	if socketName == " " {
		logger.Info("Error creating socketName")
		logger.Warn(err.Error())
		os.Exit(1)
	}

	// TODO: systemd implementation

	logger.Info("Creating systemd service")
	err = makeService(*userID, *userServerID, socketName)
	if err != nil {
		logger.Warn(err.Error())
		logger.Info("Error running makeService for", "user", socketName)
	}
	logger.Info("Systemd service created - ", "user", socketName)

	logger.Info(fmt.Sprintf("Successfully deployed user %v's architechture and service %v.service on port %v", newUser.Uid, socketName, userServerPort))
	os.Exit(1)
}