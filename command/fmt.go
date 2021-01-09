package command

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	hclutils "github.com/hashicorp/packer/hcl2template"
	"github.com/posener/complete"
)

type FormatCommand struct {
	Meta
}

func (c *FormatCommand) Run(args []string) int {
	cfg, ret := c.ParseArgs(args)
	if ret != 0 {
		return ret
	}

	return c.RunContext(cfg)
}

func (c *FormatCommand) ParseArgs(args []string) (*FormatArgs, int) {
	var cfg FormatArgs
	flags := c.Meta.FlagSet("format", FlagSetNone)
	flags.Usage = func() { c.Ui.Say(c.Help()) }
	cfg.AddFlagSets(flags)
	if err := flags.Parse(args); err != nil {
		return &cfg, 1
	}

	args = flags.Args()
	if len(args) != 1 {
		flags.Usage()
		return &cfg, 1
	}

	cfg.Path = args[0]
	return &cfg, 0
}

func (c *FormatCommand) RunContext(cla *FormatArgs) int {
	if cla.Check {
		cla.Write = false
	}

	formatter := hclutils.HCL2Formatter{
		ShowDiff: cla.Diff,
		Write:    cla.Write,
		Output:   os.Stdout,
	}

	var diags hcl.Diagnostics
	// TODO should I return something else from here?
	bytesModified := c.processDir(cla.Path, cla.Recursive, formatter, diags)
	ret := writeDiags(c.Ui, nil, diags)
	if ret != 0 {
		return ret
	}

	if cla.Check && bytesModified > 0 {
		return 3
	}

	return 0
}

// TODO determine if asterisk matters for Diagnostics here
func (c *FormatCommand) processDir(path string, recursive bool, formatter hclutils.HCL2Formatter, diags hcl.Diagnostics) int {

	// TODO put the loop here for recursion
	bytesModified, currentDiag := formatter.Format(path)
	diags = diags.Extend(currentDiag)

	if recursive {
		entries, err := ioutil.ReadDir(path)
		if err != nil {
			switch {
			case os.IsNotExist(err):
				diags = diags.Extend(fmt.Errorf("There is no configuration directory at %s", path))
			default:
				// ReadDir does not produce error messages that are end-user-appropriate,
				// so we'll need to simplify here.
				diags = diags.Extend(fmt.Errorf("Cannot read directory %s", path))
			}
		}

		for _, info := range entries {
			name := info.Name()
			// TODO determine if I need this
			// if configs.IsIgnoredFile(name) {
			//   continue
			// }
			subPath := filepath.Join(path, name)
			if info.IsDir() {
				bytesModified += c.processDir(subPath, recursive, formatter, diags)
			}
		}
	}

	return bytesModified
}

func (*FormatCommand) Help() string {
	helpText := `
Usage: packer fmt [options] [TEMPLATE]

  Rewrites all Packer configuration files to a canonical format. Both
  configuration files (.pkr.hcl) and variable files (.pkrvars.hcl) are updated.
  JSON files (.json) are not modified.

  If TEMPATE is "." the current directory will be used. The given content must
  be in Packer's HCL2 configuration language; JSON is not supported.

Options:
  -check        Check if the input is formatted. Exit status will be 0 if all
                 input is properly formatted and non-zero otherwise.

  -diff         Display diffs of formatting change

  -write=false  Don't write to source files
                (always disabled if using -check)

  -recursive    Also process files in subdirectories. By default, only the
                given directory (or current directory) is processed.
`

	return strings.TrimSpace(helpText)
}

func (*FormatCommand) Synopsis() string {
	return "Rewrites HCL2 config files to canonical format"
}

func (*FormatCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictNothing
}

func (*FormatCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-check": complete.PredictNothing,
		"-diff":  complete.PredictNothing,
		"-write": complete.PredictNothing,
	}
}
