package main

import (
    "fmt"
    "github.com/ekara-platform/engine/ssh"
    "log"
    "os"
    "path/filepath"
    "strconv"

    "github.com/ekara-platform/engine/model"
    "github.com/ekara-platform/engine/util"
)

const (
    envHTTPProxy  string = "http_proxy"
    envHTTPSProxy string = "https_proxy"
    envNoProxy    string = "no_proxy"
)

type (
    //InstallerContext Represents the informations required start the ekara engine
    // from the installer container
    installerContext struct {
        fN            util.FeedbackNotifier
        logger        *log.Logger
        skip          int
        verbosity     int
        ef            util.ExchangeFolder
        sshPublicKey  string
        sshPrivateKey string
        proxy         model.Proxy
        extVars       model.Parameters
    }
)

func (c installerContext) Feedback() util.FeedbackNotifier {
    return c.fN
}

//Log the logger to used during the ekara execution
func (c installerContext) Log() *log.Logger {
    return c.logger
}

//Skip is the level of requested skipping for the engine (0, 1 or 2).
func (c installerContext) Skipping() int {
    return c.skip
}

//Verbosity is the level of requested verbosity for the engine (0, 1 or 2).
func (c installerContext) Verbosity() int {
    return c.verbosity
}

//Ef the exchange folder used with the client machine.
func (c installerContext) Ef() util.ExchangeFolder {
    return c.ef
}

//Proxy is the proxy info used by the engine during the process execution
func (c installerContext) Proxy() model.Proxy {
    return c.proxy
}

//SSHPublicKey the public key used by the engine during the process execution to
// connect the created nodes
func (c installerContext) SSHPublicKey() string {
    return c.sshPublicKey
}

//SSHPrivateKey the private key used by the engine during the process execution to
// connect the created nodes
func (c installerContext) SSHPrivateKey() string {
    return c.sshPrivateKey
}

//ExternalVars returns the external variables passed to the installer through the CLI
func (c installerContext) ExternalVars() model.Parameters {
    return c.extVars
}

//CreateContext returns a new installer context used to run the engine
func createInstallerContext(l *log.Logger) *installerContext {
    c := &installerContext{}
    c.logger = l
    c.fN = util.CreateLoggingProgressNotifier(c.logger)
    return c
}

func fillContext(c *installerContext) error {
    fillProxy(c)
    fillVerbosity(c)
    fillSkipping(c)
    if e := fillExchangeFolder(c); e != nil {
        return e
    }
    if e := fillTemplateContext(c); e != nil {
        return e
    }
    return nil
}

// fillProxy loads the proxy settings form the environment variables into the
// context
func fillProxy(c *installerContext) {
    c.proxy = model.Proxy{
        Http:    os.Getenv(envHTTPProxy),
        Https:   os.Getenv(envHTTPSProxy),
        NoProxy: os.Getenv(envNoProxy)}
}

// fillSkipping fills the engine skipping level based on an environment variable
func fillSkipping(c *installerContext) {
    var err error
    c.skip, err = strconv.Atoi(os.Getenv(util.ActionEnvVariableSkip))
    if err != nil {
        c.skip = 0
    }
}

// fillVerbosity fills the engine verbosity level based on an environment variable
func fillVerbosity(c *installerContext) {
    var err error
    c.verbosity, err = strconv.Atoi(os.Getenv(util.StarterVerbosityVariableKey))
    if err != nil {
        c.verbosity = 2
    }
}

func fillExchangeFolder(c *installerContext) error {
    var err error
    c.ef, err = util.CreateExchangeFolder(util.InstallerVolume, "")
    if err != nil {
        return fmt.Errorf("error creating the installer exchange folder: %s", err.Error())
    }
    return nil
}

func fillTemplateContext(c *installerContext) error {
    ok := c.Ef().Location.Contains(util.ExternalVarsFilename)
    if ok {
        var e error
        c.extVars, e = model.ParseParameters(util.JoinPaths(c.Ef().Location.Path(), util.ExternalVarsFilename))
        if e != nil {
            return fmt.Errorf(errorLoadingClientParameters, e)
        }
        c.Log().Printf(logCLiParameters, c.extVars)
    }
    return nil
}

// fSHKeys checks if the SSH keys are specified via environment variables.
//
// If:
//		YES; they will be loaded into the context
//		NOT; they will be generated and then loaded into the context
//
func fillSSHKeys(c *installerContext) error {
    if c.Ef().Input.Contains(util.SSHPublicKeyFileName) && c.Ef().Input.Contains(util.SSHPrivateKeyFileName) {
        c.sshPublicKey = filepath.Join(c.Ef().Input.Path(), util.SSHPublicKeyFileName)
        c.sshPrivateKey = filepath.Join(c.Ef().Input.Path(), util.SSHPrivateKeyFileName)
        c.Log().Println("Using provided SSH keys")
    } else {
        c.Log().Println("Generating a new set of SSH keys")
        publicKey, privateKey, e := ssh.Generate()
        if e != nil {
            return fmt.Errorf(errorGeneratingSShKeys, e.Error())
        }
        c.sshPublicKey, e = util.SaveFile(c.Ef().Input, util.SSHPublicKeyFileName, publicKey)
        if e != nil {
            return fmt.Errorf("an error occurred saving the public key into: %v", c.Ef().Input.Path())
        }
        _ = os.Chmod(c.sshPublicKey, 0600)
        c.sshPrivateKey, e = util.SaveFile(c.Ef().Input, util.SSHPrivateKeyFileName, privateKey)
        if e != nil {
            return fmt.Errorf("an error occurred saving the private key into: %v", c.Ef().Input.Path())
        }
        _ = os.Chmod(c.sshPrivateKey, 0600)
    }

    if c.Log() != nil {
        c.Log().Printf(logSSHPublicKey, c.sshPublicKey)
        c.Log().Printf(logSSHPrivateKey, c.sshPrivateKey)
    }
    return nil
}
