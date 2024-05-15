package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/version"

	provconfig "github.com/provenance-io/provenance/cmd/provenanced/config"
)

const (
	// entryNotFound is a magic index value that indicates a thing wasn't found (and so no index is applicable).
	entryNotFound = -1

	// addedLeadUpdated is an added lead for a header to indicate that the section represents updates.
	addedLeadUpdated = "Updated"
	// addedLeadChanged is an added lead for a header to indicate that the section represents values different from their defaults.
	addedLeadChanged = "Differences from Defaults"
)

var configCmdStart = fmt.Sprintf("%s config", version.AppName)

// ConfigCmd returns a CLI command to update config files.
func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "config",
		Aliases:                    []string{"conf"},
		Short:                      "Manage configuration values",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		ConfigGetCmd(),
		ConfigSetCmd(),
		ConfigChangedCmd(),
		ConfigHomeCmd(),
		ConfigPackCmd(),
		ConfigUnpackCmd(),
	)
	return cmd
}

// ConfigGetCmd returns a CLI command to get config values.
func ConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [<key1> [<key2> ...]]",
		Short: "Get configuration values",
		Long: fmt.Sprintf(`Get configuration values.

    The key values can be specific.
        e.g. %[1]s get telemetry.service-name moniker.
    Or they can be parent field names
        e.g. %[1]s get api consensus
    Or they can be a type of config file:
        "cosmos", "app" -> %[2]s configuration values.
            e.g. %[1]s get app
        "cometbft", "comet", "cmt", "config" -> %[3]s configuration values.
            e.g. %[1]s get cmt
        "client" -> %[4]s configuration values.
            e.g. %[1]s get client
    Or they can be the word "all" to get all configuration values.
        e.g. %[1]s get all
    If no keys are provided, all values are retrieved.

    Displayed values will reflect settings defined through environment variables.

`, configCmdStart, provconfig.AppConfFilename, provconfig.CmtConfFilename, provconfig.ClientConfFilename),
		Example: fmt.Sprintf(`$ %[1]s get telemetry.service-name moniker \
$ %[1]s get api consensus \
$ %[1]s get app \
$ %[1]s get cmt \
$ %[1]s get client \
$ %[1]s get all \
			`, configCmdStart),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runConfigGetCmd(cmd, args)
			// Note: If a RunE returns an error, the usage information is displayed.
			//       That ends up being kind of annoying with this command.
			//       So just output the error and still return nil.
			if err != nil {
				cmd.Printf("Error: %v\n", err)
			}
			return nil
		},
	}
	return cmd
}

// ConfigSetCmd returns a CLI command to set config values.
func ConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [key1] [value1] [[<key2> <value2> ...]]",
		Short: "Set configuration values",
		Long: fmt.Sprintf(`Set configuration values.

Set a config value: %[1]s set <key> <value>
    The key must be specific, e.g. "telemetry.service-name", or "moniker".
    The value must be provided as a single, separate argument.
    e.g. %[1]s set output json

Set multiple config values %[1]s set <key1> <value1> [<key2> <value2> ...]
    Simply provide multiple key/value pairs as alternating arguments.
    e.g. %[1]s set api.enable true api.swagger true

`, configCmdStart),
		Example: fmt.Sprintf(`$ %[1]s set output json \
$ %[1]s set api.enable true api.swagger true
`, configCmdStart),
		RunE: func(cmd *cobra.Command, args []string) error {
			showHelp, err := runConfigSetCmd(cmd, args)
			// Note: If a RunE returns an error, the usage information is displayed.
			//       That ends up being kind of annoying in most cases in here.
			//       So only return the error when extra help is desired.
			if err != nil {
				if showHelp {
					return err
				}
				cmd.Printf("Error: %v\n", err)
			}
			return nil
		},
	}
	return cmd
}

// ConfigChangedCmd returns a CLI command to get config values different from their defaults.
func ConfigChangedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changed [[key1] [[key2]...]",
		Short: "Get configuration values that are different from their default.",
		Long: fmt.Sprintf(`Get configuration values that are different from their default.

Get just the configuration entries that are not default values: %[1]s changed [<key1> [<key2> ...]]
    The key values can be specific.
        e.g. %[1]s get telemetry.service-name moniker.
        Specific keys that are provided that are equal to default values will still be included in output,
            but they will be noted as such.
    Or they can be parent field names
        e.g. %[1]s get api consensus
    Or they can be a type of config file:
        "cosmos", "app" -> %[2]s configuration values.
            e.g. %[1]s get app
        "cometbft", "comet", "cmt", "config" -> %[3]s configuration values.
            e.g. %[1]s get cmt
        "client" -> %[4]s configuration values.
            e.g. %[1]s get client
    Or they can be the word "all" to get all configuration values.
        e.g. %[1]s get all
    Current and default values are both included in the output.
    If no keys are provided, all non-default values are retrieved.

    Displayed values will reflect settings defined through environment variables.

`, configCmdStart, provconfig.AppConfFilename, provconfig.CmtConfFilename, provconfig.ClientConfFilename),
		Example: fmt.Sprintf(`$ %[1]s changed \
$ %[1]s changed telemetry.service-name`, configCmdStart),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runConfigChangedCmd(cmd, args)
			// Note: If a RunE returns an error, the usage information is displayed.
			//       That ends up being kind of annoying with this command.
			//       So just output the error and still return nil.
			if err != nil {
				cmd.Printf("Error: %v\n", err)
			}
			return nil
		},
	}
	return cmd
}

// ConfigHomeCmd returns a CLI command for ouputting the home directory
func ConfigHomeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "home",
		Short: "Outputs the home directory.",
		Long: `Outputs the home directory.
		
The directory that houses the configuration and data for the blockchain. This directory can be set with either PIO_HOME or --home.
		`,
		Example: fmt.Sprintf(`$ %[1]s home`, configCmdStart),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigHomeCmd(cmd)
		},
	}
	return cmd
}

// ConfigPackCmd returns a CLI command for creating a single packed json config file.
func ConfigPackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Unpack configuration into a single config file",
		Long: fmt.Sprintf(`Unpack configuration into a single config file

Combines the %[2]s, %[3]s, and %[4]s files into %[1]s.
Settings defined through environment variables will be included in the packed file.
Settings that are their default value will not be included.

`, provconfig.PackedConfFilename, provconfig.AppConfFilename, provconfig.CmtConfFilename, provconfig.ClientConfFilename),
		Example: fmt.Sprintf(`$ %[1]s pack`, configCmdStart),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigPackCmd(cmd)
		},
	}
	return cmd
}

// ConfigUnpackCmd returns a CLI command for creating the several config toml files.
func ConfigUnpackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unpack",
		Aliases: []string{"update"},
		Short:   "Unpack configuration into separate config files",
		Long: fmt.Sprintf(`Unpack configuration into separate config files.

Splits the %[1]s file into %[2]s, %[3]s, and %[4]s.
Settings defined through environment variables will be included in the unpacked files.
Default values are filled in appropriately.

This can also be used to update the config files using the current template so they include all current fields.

`, provconfig.PackedConfFilename, provconfig.AppConfFilename, provconfig.CmtConfFilename, provconfig.ClientConfFilename),
		Example: fmt.Sprintf(`$ %[1]s unpack`, configCmdStart),
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigUnpackCmd(cmd)
		},
	}
	return cmd
}

// runConfigGetCmd gets requested values and outputs them.
func runConfigGetCmd(cmd *cobra.Command, args []string) error {
	_, appFields, acerr := provconfig.ExtractAppConfigAndMap(cmd)
	if acerr != nil {
		return fmt.Errorf("could not get app config fields: %w", acerr)
	}
	_, cmtFields, cmtcerr := provconfig.ExtractCmtConfigAndMap(cmd)
	if cmtcerr != nil {
		return fmt.Errorf("could not get cometbft config fields: %w", cmtcerr)
	}
	_, clientFields, ccerr := provconfig.ExtractClientConfigAndMap(cmd)
	if ccerr != nil {
		return fmt.Errorf("could not get client config fields: %w", ccerr)
	}

	if len(args) == 0 {
		args = append(args, "all")
	}

	appToOutput := provconfig.FieldValueMap{}
	cmtToOutput := provconfig.FieldValueMap{}
	clientToOutput := provconfig.FieldValueMap{}
	unknownKeyMap := provconfig.FieldValueMap{}
	for _, key := range args {
		switch key {
		case "all":
			appToOutput.AddEntriesFrom(appFields)
			cmtToOutput.AddEntriesFrom(cmtFields)
			clientToOutput.AddEntriesFrom(clientFields)
		case "app", "cosmos":
			appToOutput.AddEntriesFrom(appFields)
		case "tendermint", "tm":
			cmd.Printf("The %q option is deprecated and will be removed in a future version.\n", key)
			cmd.Println("Use one of \"cometbft\", \"comet\", or \"cmt\" instead.")
			fallthrough
		case "config", "cometbft", "comet", "cmt":
			cmtToOutput.AddEntriesFrom(cmtFields)
		case "client":
			clientToOutput.AddEntriesFrom(clientFields)
		default:
			appFVM, appFound, appExact := appFields.FindEntries(key)
			cmtFVM, cmtFound, cmtExact := cmtFields.FindEntries(key)
			clientFVM, clientFound, clientExact := clientFields.FindEntries(key)

			found := appFound || cmtFound || clientFound
			if !found {
				unknownKeyMap.SetToNil(key)
				continue
			}

			haveExact := appExact || cmtExact || clientExact
			if appFound && (!haveExact || appExact) {
				appToOutput.AddEntriesFrom(appFVM)
			}
			if cmtFound && (!haveExact || cmtExact) {
				cmtToOutput.AddEntriesFrom(cmtFVM)
			}
			if clientFound && (!haveExact || clientExact) {
				clientToOutput.AddEntriesFrom(clientFVM)
			}
		}
	}

	isPacked := provconfig.IsPacked(cmd)
	if len(appToOutput) > 0 {
		cmd.Println(makeAppConfigHeader(cmd, "", isPacked).String())
		cmd.Println(makeFieldMapString(appToOutput))
	}
	if len(cmtToOutput) > 0 {
		cmd.Println(makeCmtConfigHeader(cmd, "", isPacked).String())
		cmd.Println(makeFieldMapString(cmtToOutput))
	}
	if len(clientToOutput) > 0 {
		cmd.Println(makeClientConfigHeader(cmd, "", isPacked).String())
		cmd.Println(makeFieldMapString(clientToOutput))
	}
	if isPacked && (len(appToOutput) > 0 || len(cmtToOutput) > 0 || len(clientToOutput) > 0) {
		cmd.Println(makeConfigIsPackedLine(cmd))
	}
	if len(unknownKeyMap) > 0 {
		unknownKeys := unknownKeyMap.GetSortedKeys()
		s := "s"
		if len(unknownKeys) == 1 {
			s = ""
		}
		return fmt.Errorf("%d configuration key%s not found: %s", len(unknownKeys), s, strings.Join(unknownKeys, ", "))
	}
	return nil
}

// runConfigSetCmd sets values as provided.
// The first return value is whether to include help with the output of an error.
// This will only ever be true if an error is also returned.
// The second return value is any error encountered.
func runConfigSetCmd(cmd *cobra.Command, args []string) (bool, error) {
	if len(args) == 0 {
		return true, errors.New("no key/value pairs provided")
	}
	if len(args)%2 != 0 {
		return true, errors.New("an even number of arguments are required when setting values")
	}

	// Warning: This wipes out all the viper setup stuff up to this point.
	// It needs to be done so that just the file values or defaults are loaded
	// without considering environment variables.
	clientCtx := client.GetClientContextFromCmd(cmd)
	clientCtx.Viper = viper.New()
	server.GetServerContextFromCmd(cmd).Viper = clientCtx.Viper
	if err := client.SetCmdClientContext(cmd, clientCtx); err != nil {
		return false, err
	}

	// Now that we have a clean viper, load the config from files again.
	if err := provconfig.LoadConfigFromFiles(cmd); err != nil {
		return false, err
	}

	appConfig, appFields, acerr := provconfig.ExtractAppConfigAndMap(cmd)
	if acerr != nil {
		return false, fmt.Errorf("couldn't get app config: %w", acerr)
	}
	cmtConfig, cmtFields, cmtcerr := provconfig.ExtractCmtConfigAndMap(cmd)
	if cmtcerr != nil {
		return false, fmt.Errorf("couldn't get cometbft config: %w", cmtcerr)
	}
	clientConfig, clientFields, ccerr := provconfig.ExtractClientConfigAndMap(cmd)
	if ccerr != nil {
		return false, fmt.Errorf("couldn't get client config: %w", ccerr)
	}

	keyCount := len(args) / 2
	keys := make([]string, keyCount)
	vals := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = args[i*2]
		vals[i] = args[i*2+1]
	}
	issueFound := false
	appUpdates := provconfig.UpdatedFieldMap{}
	cmtUpdates := provconfig.UpdatedFieldMap{}
	clientUpdates := provconfig.UpdatedFieldMap{}
	for i, key := range keys {
		var confMap provconfig.FieldValueMap
		foundIn := entryNotFound
		for fvmi, fvm := range []provconfig.FieldValueMap{appFields, cmtFields, clientFields} {
			if fvm.Has(key) {
				confMap = fvm
				foundIn = fvmi
				break
			}
		}
		if foundIn == entryNotFound {
			cmd.Printf("Configuration key %s does not exist.\n", key)
			issueFound = true
			continue
		}
		was := confMap.GetStringOf(key)
		err := confMap.SetFromString(key, vals[i])
		if err != nil {
			cmd.Printf("Error setting key %s: %v\n", key, err)
			issueFound = true
			continue
		}
		isNow := confMap.GetStringOf(key)
		switch foundIn {
		case 0:
			appUpdates.AddOrUpdate(key, was, isNow)
		case 1:
			cmtUpdates.AddOrUpdate(key, was, isNow)
		case 2:
			clientUpdates.AddOrUpdate(key, was, isNow)
		}
	}
	if !issueFound {
		if len(appUpdates) > 0 {
			if err := appConfig.ValidateBasic(); err != nil {
				cmd.Printf("App config validation error: %v\n", err)
				issueFound = true
			}
		}
		if len(cmtUpdates) > 0 {
			if err := cmtConfig.ValidateBasic(); err != nil {
				cmd.Printf("CometBFT config validation error: %v\n", err)
				issueFound = true
			}
		}
		if len(clientUpdates) > 0 {
			if err := clientConfig.ValidateBasic(); err != nil {
				cmd.Printf("Client config validation error: %v\n", err)
				issueFound = true
			}
		}
	}
	if issueFound {
		return false, errors.New("one or more issues encountered; no configuration values have been updated")
	}
	// If a certain config hasn't been changed, we want to provide it as nil to the SaveConfigs func.
	if len(appUpdates) == 0 {
		appConfig = nil
	}
	if len(cmtUpdates) == 0 {
		cmtConfig = nil
	}
	if len(clientUpdates) == 0 {
		clientConfig = nil
	}
	provconfig.SaveConfigs(cmd, appConfig, cmtConfig, clientConfig, false)
	isPacked := provconfig.IsPacked(cmd)
	if len(appUpdates) > 0 {
		cmd.Println(makeAppConfigHeader(cmd, addedLeadUpdated, isPacked).WithoutEnv().String())
		cmd.Println(makeUpdatedFieldMapString(appUpdates, provconfig.UpdatedField.StringAsUpdate))
	}
	if len(cmtUpdates) > 0 {
		cmd.Println(makeCmtConfigHeader(cmd, addedLeadUpdated, isPacked).WithoutEnv().String())
		cmd.Println(makeUpdatedFieldMapString(cmtUpdates, provconfig.UpdatedField.StringAsUpdate))
	}
	if len(clientUpdates) > 0 {
		cmd.Println(makeClientConfigHeader(cmd, addedLeadUpdated, isPacked).WithoutEnv().String())
		cmd.Println(makeUpdatedFieldMapString(clientUpdates, provconfig.UpdatedField.StringAsUpdate))
	}
	if isPacked && (len(appUpdates) > 0 || len(cmtUpdates) > 0 || len(clientUpdates) > 0) {
		cmd.Println(makeConfigIsPackedLine(cmd))
	}
	return false, nil
}

// runConfigChangedCmd gets values that have changed from their defaults.
func runConfigChangedCmd(cmd *cobra.Command, args []string) error {
	_, appFields, acerr := provconfig.ExtractAppConfigAndMap(cmd)
	if acerr != nil {
		return fmt.Errorf("couldn't get app config: %w", acerr)
	}
	_, cmtFields, cmtcerr := provconfig.ExtractCmtConfigAndMap(cmd)
	if cmtcerr != nil {
		return fmt.Errorf("couldn't get cometbft config: %w", cmtcerr)
	}
	_, clientFields, ccerr := provconfig.ExtractClientConfigAndMap(cmd)
	if ccerr != nil {
		return fmt.Errorf("couldn't get client config: %w", ccerr)
	}

	if len(args) == 0 {
		args = append(args, "all")
	}

	allDefaults := provconfig.GetAllConfigDefaults()
	showApp, showCmt, showClient := false, false, false
	appDiffs := provconfig.UpdatedFieldMap{}
	cmtDiffs := provconfig.UpdatedFieldMap{}
	clientDiffs := provconfig.UpdatedFieldMap{}
	unknownKeyMap := provconfig.FieldValueMap{}
	for _, key := range args {
		switch key {
		case "all":
			showApp, showCmt, showClient = true, true, true
			appDiffs.AddOrUpdateEntriesFrom(provconfig.MakeUpdatedFieldMap(allDefaults, appFields, true))
			cmtDiffs.AddOrUpdateEntriesFrom(provconfig.MakeUpdatedFieldMap(allDefaults, cmtFields, true))
			clientDiffs.AddOrUpdateEntriesFrom(provconfig.MakeUpdatedFieldMap(allDefaults, clientFields, true))
		case "app", "cosmos":
			showApp = true
			appDiffs.AddOrUpdateEntriesFrom(provconfig.MakeUpdatedFieldMap(allDefaults, appFields, true))
		case "tendermint", "tm":
			cmd.Printf("The %q option is deprecated and will be removed in a future version.\n", key)
			cmd.Println("Use one of \"cometbft\", \"comet\", or \"cmt\" instead.")
			fallthrough
		case "config", "cometbft", "comet", "cmt":
			showCmt = true
			cmtDiffs.AddOrUpdateEntriesFrom(provconfig.MakeUpdatedFieldMap(allDefaults, cmtFields, true))
		case "client":
			showClient = true
			clientDiffs.AddOrUpdateEntriesFrom(provconfig.MakeUpdatedFieldMap(allDefaults, clientFields, true))
		default:
			appFVM, appFound, appExact := appFields.FindEntries(key)
			cmtFVM, cmtFound, cmtExact := cmtFields.FindEntries(key)
			clientFVM, clientFound, clientExact := clientFields.FindEntries(key)

			found := appFound || cmtFound || clientFound
			if !found {
				unknownKeyMap.SetToNil(key)
				continue
			}

			haveExact := appExact || cmtExact || clientExact
			if appFound && (!haveExact || appExact) {
				showApp = true
				changes := provconfig.MakeUpdatedFieldMap(allDefaults, appFVM, false)
				appDiffs.AddOrUpdateEntriesFrom(changes)
			}
			if cmtFound && (!haveExact || cmtExact) {
				showCmt = true
				changes := provconfig.MakeUpdatedFieldMap(allDefaults, cmtFVM, false)
				cmtDiffs.AddOrUpdateEntriesFrom(changes)
			}
			if clientFound && (!haveExact || clientExact) {
				showClient = true
				changes := provconfig.MakeUpdatedFieldMap(allDefaults, clientFVM, false)
				clientDiffs.AddOrUpdateEntriesFrom(changes)
			}
		}
	}

	isPacked := provconfig.IsPacked(cmd)

	if showApp {
		cmd.Println(makeAppConfigHeader(cmd, addedLeadChanged, isPacked).String())
		if len(appDiffs) > 0 {
			cmd.Println(makeUpdatedFieldMapString(appDiffs, provconfig.UpdatedField.StringAsDefault))
		} else {
			cmd.Println("All app config values equal the default config values.")
			cmd.Println("")
		}
	}

	if showCmt {
		cmd.Println(makeCmtConfigHeader(cmd, addedLeadChanged, isPacked).String())
		if len(cmtDiffs) > 0 {
			cmd.Println(makeUpdatedFieldMapString(cmtDiffs, provconfig.UpdatedField.StringAsDefault))
		} else {
			cmd.Println("All cometbft config values equal the default config values.")
			cmd.Println("")
		}
	}

	if showClient {
		cmd.Println(makeClientConfigHeader(cmd, addedLeadChanged, isPacked).String())
		if len(clientDiffs) > 0 {
			cmd.Println(makeUpdatedFieldMapString(clientDiffs, provconfig.UpdatedField.StringAsDefault))
		} else {
			cmd.Println("All client config values equal the default config values.")
			cmd.Println("")
		}
	}

	if isPacked && (showApp || showCmt || showClient) {
		cmd.Println(makeConfigIsPackedLine(cmd))
	}

	if len(unknownKeyMap) > 0 {
		unknownKeys := unknownKeyMap.GetSortedKeys()
		s := "s"
		if len(unknownKeys) == 1 {
			s = ""
		}
		return fmt.Errorf("%d configuration key%s not found: %s", len(unknownKeys), s, strings.Join(unknownKeys, ", "))
	}
	return nil
}

// runConfigHomeCmd obtains the home directory.
func runConfigHomeCmd(cmd *cobra.Command) error {
	clientCtx := client.GetClientContextFromCmd(cmd)
	cmd.Println(clientCtx.HomeDir)
	return nil
}

// runConfigPackCmd combines the toml config files into a single config json file.
func runConfigPackCmd(cmd *cobra.Command) error {
	return provconfig.PackConfig(cmd)
}

// runConfigUnpackCmd converts a single config json file into the individual toml files.
func runConfigUnpackCmd(cmd *cobra.Command) error {
	return provconfig.UnpackConfig(cmd)
}

// makeFieldMapString makes a multi-line string with all the keys and values in the provided map.
func makeFieldMapString(m provconfig.FieldValueMap) string {
	keys := m.GetSortedKeys()
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(m.GetStringOf(k))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// makeUpdatedFieldMapString makes a multi-line string of the given updated field map.
// The provided stringer function is used to convert each map value to a string.
func makeUpdatedFieldMapString(m provconfig.UpdatedFieldMap, stringer func(v provconfig.UpdatedField) string) string {
	keys := m.GetSortedKeys()
	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(stringer(*m[key]))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// sectionHeader is a struct holding several options for section header strings.
type sectionHeader struct {
	lead      string
	addedLead string
	filename  string
	isPacked  bool
	env       bool
}

// WithoutEnv sets env to false returning itself.
func (s *sectionHeader) WithoutEnv() *sectionHeader {
	s.env = false
	return s
}

// Create the section header string desired.
func (s sectionHeader) String() string {
	var sb strings.Builder
	sb.WriteString(s.lead)
	if len(s.addedLead) > 0 {
		sb.WriteByte(' ')
		sb.WriteString(s.addedLead)
	}
	sb.WriteByte(':')
	hr := strings.Repeat("-", sb.Len())
	if len(s.filename) > 0 {
		sb.WriteByte(' ')
		switch {
		case s.isPacked:
			sb.WriteString("(packed)")
		case !provconfig.FileExists(s.filename):
			sb.WriteString("(defaults)")
		default:
			sb.WriteString(s.filename)
		}
		if s.env {
			sb.WriteString(" (or env)")
		}
		hr += "-----"
	}
	sb.WriteByte('\n')
	sb.WriteString(hr)
	return sb.String()
}

// makeAppConfigHeader creates a section header string for app config stuff.
func makeAppConfigHeader(cmd *cobra.Command, addedLead string, isPacked bool) *sectionHeader {
	return &sectionHeader{
		lead:      "App Config",
		addedLead: addedLead,
		filename:  provconfig.GetFullPathToAppConf(cmd),
		isPacked:  isPacked,
		env:       true,
	}
}

// makeCmtConfigHeader creates a section header string for cometbft config stuff.
func makeCmtConfigHeader(cmd *cobra.Command, addedLead string, isPacked bool) *sectionHeader {
	return &sectionHeader{
		lead:      "CometBFT Config",
		addedLead: addedLead,
		filename:  provconfig.GetFullPathToCmtConf(cmd),
		isPacked:  isPacked,
		env:       true,
	}
}

// makeClientConfigHeader creates a section header string for client config stuff.
func makeClientConfigHeader(cmd *cobra.Command, addedLead string, isPacked bool) *sectionHeader {
	return &sectionHeader{
		lead:      "Client Config",
		addedLead: addedLead,
		filename:  provconfig.GetFullPathToClientConf(cmd),
		isPacked:  isPacked,
		env:       true,
	}
}

// makeConfigIsPackedLine creates a line indicating that the config is packed (and where to find it).
func makeConfigIsPackedLine(cmd *cobra.Command) string {
	return fmt.Sprintf("Config is packed: %s\n", provconfig.GetFullPathToPackedConf(cmd))
}
