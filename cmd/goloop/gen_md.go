package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/icon-project/goloop/common/log"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewGenerateMarkdownCommand() *cobra.Command {
	docCmd := &cobra.Command{
		Use:   "doc [FILE]",
		Short: "generate markdown for CommandLineInterface",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			rootCmd := cmd.Root()
			filePath := rootCmd.Name() + ".md"
			if len(args) > 0 {
				filePath = args[0]
			}
			f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				log.Panicf("Fail to open file %s err=%+v", filePath, err)
			}
			GenerateMarkdown(rootCmd, f)
			f.Close()
			log.Println("generated markdown", filePath)
		},
	}
	return docCmd
}

func isIgnoreCommand(cmd *cobra.Command) bool {
	if cmd.Name() == "help" || cmd.Hidden {
		return true
	}
	return false
}

func GenerateMarkdown(cmd *cobra.Command, w io.Writer) {
	if isIgnoreCommand(cmd) {
		return
	}

	buf := new(bytes.Buffer)
	if !cmd.HasParent() {
		name := cmd.Name()
		name = strings.ToUpper(name[:1]) + name[1:]
		buf.WriteString(fmt.Sprintln("#", name))
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintln("##", cmd.CommandPath()))
	buf.WriteString("\n")

	buf.WriteString(fmt.Sprintln("###", "Description"))
	if cmd.Long == "" {
		buf.WriteString(fmt.Sprintln(cmd.Short))
	} else {
		buf.WriteString(fmt.Sprintln(cmd.Long))
	}
	buf.WriteString("\n")

	buf.WriteString(fmt.Sprintln("###", "Usage"))
	buf.WriteString(fmt.Sprintln("`", cmd.UseLine(), "`"))
	buf.WriteString("\n")

	if cmd.HasLocalFlags() || cmd.HasPersistentFlags() {
		buf.WriteString(fmt.Sprintln("###", "Options"))
		buf.WriteString(fmt.Sprintln("|Name,shorthand | Default | Description|"))
		buf.WriteString(fmt.Sprintln("|---|---|---|"))
		cmd.NonInheritedFlags().VisitAll(FlagToMarkdown(buf))

		buf.WriteString("\n")
	}

	if cmd.HasInheritedFlags() {
		buf.WriteString(fmt.Sprintln("###", "Inherited Options"))
		buf.WriteString(fmt.Sprintln("|Name,shorthand | Default | Description|"))
		buf.WriteString(fmt.Sprintln("|---|---|---|"))
		cmd.InheritedFlags().VisitAll(FlagToMarkdown(buf))
		buf.WriteString("\n")
	}

	if cmd.HasAvailableSubCommands() {
		buf.WriteString(fmt.Sprintln("###", "Child commands"))
		buf.WriteString(fmt.Sprintln("|Command | Description|"))
		buf.WriteString(fmt.Sprintln("|---|---|"))
		for _, childCmd := range cmd.Commands() {
			CommandPathToMarkdown(buf, childCmd)
		}
		buf.WriteString("\n")
	}

	if cmd.HasParent() {
		buf.WriteString(fmt.Sprintln("###", "Parent command"))
		buf.WriteString(fmt.Sprintln("|Command | Description|"))
		buf.WriteString(fmt.Sprintln("|---|---|"))
		CommandPathToMarkdown(buf, cmd.Parent())
		buf.WriteString("\n")

		buf.WriteString(fmt.Sprintln("###", "Related commands"))
		buf.WriteString(fmt.Sprintln("|Command | Description|"))
		buf.WriteString(fmt.Sprintln("|---|---|"))
		for _, childCmd := range cmd.Parent().Commands() {
			CommandPathToMarkdown(buf, childCmd)
		}
		buf.WriteString("\n")
	}
	_, _ = buf.WriteTo(w)

	if cmd.HasAvailableSubCommands() {
		for _, childCmd := range cmd.Commands() {
			GenerateMarkdown(childCmd, w)
		}
	}
}

func CommandPathToMarkdown(buf *bytes.Buffer, cmd *cobra.Command) {
	if isIgnoreCommand(cmd) {
		return
	}
	cPath := cmd.CommandPath()
	cPath = fmt.Sprintf("[%s](#%s)", cPath, strings.ReplaceAll(cPath, " ", "-"))
	buf.WriteString(fmt.Sprintln("|", cPath, "|", cmd.Short, "|"))
}

func FlagToMarkdown(buf *bytes.Buffer) func(f *pflag.Flag) {
	return func(f *pflag.Flag) {
		name := ""
		if f.Shorthand != "" && f.ShorthandDeprecated == "" {
			name = fmt.Sprintf("--%s, -%s", f.Name, f.Shorthand)
		} else {
			name = fmt.Sprintf("--%s", f.Name)
		}
		buf.WriteString(fmt.Sprintln("|", name, "|", f.DefValue, "|", f.Usage, "|"))
	}
}
