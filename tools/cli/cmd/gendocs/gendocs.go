package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kyma-project/control-plane/tools/cli/pkg/command"
	"github.com/kyma-project/control-plane/tools/cli/pkg/logger"
)

const (
	docsTargetDir = "../../docs/cli/commands"
	fmTemplate    = `# %s
`
)

func main() {
	log := logger.New()
	cmd := command.New(log)

	err := genMarkdownTree(cmd, docsTargetDir)
	if err != nil {
		fmt.Println("unable to generate docs", err.Error())
		os.Exit(1)
	}

	fmt.Println("Docs successfully generated to the following dir", docsTargetDir)
	os.Exit(0)
}

func genMarkdownTree(cmd *cobra.Command, dir string) error {
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		if err := genMarkdownTree(c, dir); err != nil {
			return err
		}
	}

	basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".md"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(cmd)); err != nil {
		return err
	}
	if err := genMarkdown(cmd, f); err != nil {
		return err
	}
	return nil
}

func genMarkdown(cmd *cobra.Command, w io.Writer) error {
	cmd.InitDefaultHelpCmd()
	cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)

	printShort(buf, cmd)
	printSynopsis(buf, cmd)

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```bash\n%s\n```\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("## Examples\n\n")
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.Example))
	}

	if err := printOptions(buf, cmd); err != nil {
		return err
	}

	printSeeAlso(buf, cmd)

	_, err := buf.WriteTo(w)
	return err
}

func printShort(buf *bytes.Buffer, cmd *cobra.Command) {
	short := cmd.Short
	buf.WriteString(short + "\n\n")
}

func printSynopsis(buf *bytes.Buffer, cmd *cobra.Command) {
	short := cmd.Short
	long := cmd.Long
	if len(long) == 0 {
		long = short
	}

	buf.WriteString("## Synopsis\n\n")
	buf.WriteString(markdownFormatCLIParts(long) + "\n\n")
}

func printOptions(buf *bytes.Buffer, cmd *cobra.Command) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("## Options\n\n```\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("## Global Options\n\n```\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	return nil
}

func printSeeAlso(buf *bytes.Buffer, cmd *cobra.Command) {
	if !hasSeeAlso(cmd) {
		return
	}

	name := cmd.CommandPath()

	buf.WriteString("## See also\n\n")
	if cmd.HasParent() {
		parent := cmd.Parent()
		pname := parent.CommandPath()
		buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", pname, linkHandler(parent), parent.Short))
		cmd.VisitParents(func(c *cobra.Command) {
			if c.DisableAutoGenTag {
				cmd.DisableAutoGenTag = c.DisableAutoGenTag
			}
		})
	}

	children := cmd.Commands()
	sort.Sort(byName(children))

	for _, child := range children {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}
		cname := name + " " + child.Name()
		buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", cname, linkHandler(child), child.Short))
	}
	buf.WriteString("\n")
}

func markdownFormatCLIParts(s string) string {
	opts := regexp.MustCompile(`--[a-zA-z0-9-_]+( \{[a-zA-z0-9-_]+\}){0,1}`)
	formatted := opts.ReplaceAllString(s, "`$0`")
	return formatted
}

func filePrepender(cmd *cobra.Command) string {
	name := cmd.CommandPath()
	return fmt.Sprintf(fmTemplate, name)
}

func linkHandler(cmd *cobra.Command) string {
	filename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".md"
	return filename
}

func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

type byName []*cobra.Command

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
