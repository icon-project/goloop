package cli

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/jroimartin/gocui"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

const (
	FlagAnnotationCustom = "custom"
)

func NewCommand(parentCmd *cobra.Command, parentVc *viper.Viper, use, short string) (*cobra.Command, *viper.Viper) {
	c := &cobra.Command{Use: use, Short: short}
	c.SetFlagErrorFunc(DefaultFlagErrorFunc)
	if parentCmd != nil {
		parentCmd.AddCommand(c)
	}

	var pFlags *pflag.FlagSet
	envPrefix := strings.ReplaceAll(c.CommandPath(), " ", "_")
	if parentVc != nil {
		if v := parentVc.Get("env_prefix"); v != nil {
			envPrefix = v.(string)
		}
		if v := parentVc.Get("pflags"); v != nil {
			pFlags = v.(*pflag.FlagSet)
		}
	}
	vc := NewViper(envPrefix)
	if pFlags != nil {
		BindPFlags(vc, pFlags)
	}
	if parentVc != nil {
		vc.Set("_parentvc_", parentVc)
		parentVc.Set("_childvc_"+c.Name(), vc)
	} else {
		viper.Set("_globalvc_"+c.CommandPath(), vc)
	}

	return c, vc
}

func getViper(cmd *cobra.Command, parentVc *viper.Viper) *viper.Viper {
	if parentVc != nil {
		if vc, ok := parentVc.Get("_childvc_" + cmd.Name()).(*viper.Viper); ok {
			return vc
		}
	}
	if vc, ok := viper.Get("_globalvc_" + cmd.CommandPath()).(*viper.Viper); ok {
		return vc
	}
	return nil
}

func getVipers(vc *viper.Viper) []*viper.Viper {
	vcs := make([]*viper.Viper, 0)
	if vc != nil {
		if parentVc, ok := vc.Get("_parentvc_").(*viper.Viper); ok {
			vcs = append(getVipers(parentVc), vc)
		} else {
			vcs = append(vcs, vc)
		}
	}
	return vcs
}

func NewViper(envPrefix string) *viper.Viper {
	vc := viper.New()
	vc.AutomaticEnv()
	vc.SetEnvPrefix(envPrefix)
	vc.Set("env_prefix", envPrefix)
	return vc
}

func BindPFlags(vc *viper.Viper, pFlags *pflag.FlagSet) error {
	var bindPFlags *pflag.FlagSet
	if v := vc.Get("pflags"); v != nil {
		bindPFlags = v.(*pflag.FlagSet)
	} else {
		bindPFlags = pflag.NewFlagSet("pflags", pflag.ContinueOnError)
		vc.Set("pflags", bindPFlags)
	}
	bindPFlags.AddFlagSet(pFlags)

	return vc.BindPFlags(pFlags)
}

func MarkAnnotationCustom(fs *pflag.FlagSet, names ...string) error {
	for _, name := range names {
		if err := fs.SetAnnotation(name, cobra.BashCompCustom, []string{FlagAnnotationCustom}); err != nil {
			return err
		}
	}
	return nil
}

func MarkAnnotationHidden(fs *pflag.FlagSet, names ...string) error {
	for _, name := range names {
		if err := fs.MarkHidden(name); err != nil {
			return err
		}
	}
	return nil
}

func MarkAnnotationRequired(fs *pflag.FlagSet, names ...string) error {
	for _, name := range names {
		if err := fs.SetAnnotation(name, cobra.BashCompOneRequiredFlag, []string{"true"}); err != nil {
			return err
		}
	}
	return nil
}

func ValidateFlags(fs *pflag.FlagSet, flagNames ...string) error {
	missingFlagNames := []string{}
	fs.VisitAll(func(f *pflag.Flag) {
		requiredAnnotation, found := f.Annotations[cobra.BashCompOneRequiredFlag]
		if found && (requiredAnnotation[0] == "true") && !f.Changed {
			missingFlagNames = append(missingFlagNames, f.Name)
		} else {
			for _, fn := range flagNames {
				if f.Name == fn {
					if !f.Changed {
						missingFlagNames = append(missingFlagNames, f.Name)
					}
					return
				}
			}
		}
	})

	if len(missingFlagNames) > 0 {
		return errors.Errorf(`required flag(s) "%s" not set`, strings.Join(missingFlagNames, `", "`))
	}
	return nil
}

func CheckFlagsWithViper(vc *viper.Viper, fs *pflag.FlagSet, flagNames ...string) error {
	missing := []string{}
	for _, name := range flagNames {
		flag := fs.Lookup(name)
		if flag.Changed == false && vc.GetString(name) == flag.DefValue {
			missing = append(missing, flag.Name)
		}
	}
	if len(missing) > 0 {

		return errors.Errorf("MissingFlags(%s)", strings.Join(missing, ","))
	}
	return nil
}

func ValidateFlagsWithViper(vc *viper.Viper, fs *pflag.FlagSet, flagNames ...string) error {
	missingFlagNames := []string{}
	fs.VisitAll(func(f *pflag.Flag) {
		check := false
		if anns, ok := f.Annotations[cobra.BashCompCustom]; ok && (anns[0] == FlagAnnotationCustom) {
			check = true
		} else {
			for _, fn := range flagNames {
				if f.Name == fn {
					check = true
					break
				}
			}
		}

		if check && !f.Changed {
			if v := vc.GetString(f.Name); v == f.DefValue {
				missingFlagNames = append(missingFlagNames, f.Name)
			}
		}
	})

	if len(missingFlagNames) > 0 {
		return errors.Errorf(`required flag(s) "%s" not set`, strings.Join(missingFlagNames, `", "`))
	}
	return nil
}

func GetStringMap(f *pflag.Flag) (map[string]interface{}, error) {
	return stringToStringConv(f.Value.String())
}

func stringToStringConv(val string) (map[string]interface{}, error) {
	if strings.HasPrefix(val, "{") {
		out := make(map[string]interface{})
		if err := json.Unmarshal([]byte(val), &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	if strings.HasPrefix(val, "[") {
		val = strings.Trim(val, "[]")
	}
	// An empty string would cause an empty map
	if len(val) == 0 {
		return map[string]interface{}{}, nil
	}
	r := csv.NewReader(strings.NewReader(val))
	ss, err := r.Read()
	if err != nil {
		return nil, err
	}
	out := make(map[string]interface{}, len(ss))
	for _, pair := range ss {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("%s must be formatted as key=value", pair)
		}
		s := kv[1]
		if iv, err := strconv.ParseInt(s, 0, 64); err == nil {
			out[kv[0]] = iv
		} else if fv, err := strconv.ParseFloat(s, 64); err == nil {
			out[kv[0]] = fv
		} else {
			out[kv[0]] = s
		}
	}
	return out, nil
}

func ViperDecodeOptJson(c *mapstructure.DecoderConfig) {
	c.TagName = "json"
	c.DecodeHook = mapstructure.ComposeDecodeHookFunc(
		func(inputValType reflect.Type, outValType reflect.Type, input interface{}) (interface{}, error) {
			if outValType.Name() == "RawMessage" {
				if inputValType.Kind() == reflect.Map && inputValType.Key().Kind() == reflect.String {
					return json.Marshal(input)
				} else if inputValType.Kind() == reflect.String && input != "" {
					return ioutil.ReadFile(input.(string))
				}
			} else if inputValType.Kind() == reflect.String && outValType.Kind() == reflect.Map {
				m, err := stringToStringConv(input.(string))
				if outValType.Key().Kind() == reflect.String && outValType.Elem().Name() == "RawMessage" {
					m2 := make(map[string]json.RawMessage)
					for k, v := range m {
						if s, ok := v.(string); ok {
							m2[k] = json.RawMessage(s)
						}
					}
					return m2, nil
				}
				return m, err
			}
			return input, nil
		},
		c.DecodeHook)
}

func OrArgs(pArgs ...cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		var err error
		for _, pArg := range pArgs {
			if err = pArg(cmd, args); err == nil {
				return nil
			}
		}
		return err
	}
}

func ArgsWithErrorFunc(arg cobra.PositionalArgs,
	errFunc func(cmd *cobra.Command, err error) error) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := arg(cmd, args); err != nil {
			return errFunc(cmd, err)
		}
		return nil
	}
}

func ArgsWithDefaultErrorFunc(arg cobra.PositionalArgs) cobra.PositionalArgs {
	return ArgsWithErrorFunc(arg, DefaultArgErrorFunc)
}
func DefaultArgErrorFunc(cmd *cobra.Command, err error) error {
	cmd.Println("Usage: " + cmd.UseLine())
	return err
}

func DefaultFlagErrorFunc(cmd *cobra.Command, err error) error {
	names := make([]string, 0)
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		name := "--" + f.Name
		if f.Shorthand != "" {
			name = name + " or -" + f.Shorthand
		}
		names = append(names, name)
	})
	cmd.Println("Available Flags: " + strings.Join(names, ", "))
	return err
}

func NewGenerateMarkdownCommand(parentCmd *cobra.Command, parentVc *viper.Viper) *cobra.Command {
	rootCmd := &cobra.Command{Use: "doc FILE", Short: "generate markdown for CommandLineInterface"}
	if parentCmd != nil {
		parentCmd.AddCommand(rootCmd)
	}
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		filePath := cmd.Root().Name() + ".md"
		if len(args) > 0 {
			filePath = args[0]
		}
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		GenerateMarkdown(cmd.Root(), parentVc, f)
		return nil
	}
	return rootCmd
}

func isIgnoreCommand(cmd *cobra.Command) bool {
	if cmd.Name() == "help" || cmd.Hidden {
		return true
	}
	return false
}

func GenerateMarkdown(cmd *cobra.Command, parentVc *viper.Viper, w io.Writer) {
	if isIgnoreCommand(cmd) {
		return
	}

	vc := getViper(cmd, parentVc)

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
	if cmd.Deprecated != "" {
		buf.WriteString(fmt.Sprintln(fmt.Sprintf("Command %q is deprecated, %s\n", cmd.Name(), cmd.Deprecated)))
	}
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
		buf.WriteString(fmt.Sprintln("|Name,shorthand | Environment Variable | Required | Default | Description|"))
		buf.WriteString(fmt.Sprintln("|---|---|---|---|---|"))
		cmd.NonInheritedFlags().VisitAll(FlagToMarkdown(buf, vc))
		buf.WriteString("\n")
	}

	if cmd.HasInheritedFlags() {
		buf.WriteString(fmt.Sprintln("###", "Inherited Options"))
		buf.WriteString(fmt.Sprintln("|Name,shorthand | Environment Variable | Required | Default | Description|"))
		buf.WriteString(fmt.Sprintln("|---|---|---|---|---|"))
		cmd.InheritedFlags().VisitAll(FlagToMarkdown(buf, getVipers(parentVc)...))
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
			GenerateMarkdown(childCmd, vc, w)
		}
	}
}

func CommandPathToMarkdown(buf *bytes.Buffer, cmd *cobra.Command) {
	if isIgnoreCommand(cmd) {
		return
	}
	cPath := cmd.CommandPath()
	cPath = fmt.Sprintf("[%s](#%s)", cPath, strings.ReplaceAll(cPath, " ", "-"))
	deprecated := ""
	if cmd.Deprecated != "" {
		deprecated = "[deprecated]"
	}
	buf.WriteString(fmt.Sprintln("|", cPath, "|", deprecated, cmd.Short, "|"))
}

func FlagToMarkdown(buf *bytes.Buffer, vcs ...*viper.Viper) func(f *pflag.Flag) {
	return func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		name := ""
		if f.Shorthand != "" && f.ShorthandDeprecated == "" {
			name = fmt.Sprintf("--%s, -%s", f.Name, f.Shorthand)
		} else {
			name = fmt.Sprintf("--%s", f.Name)
		}
		envKey := ""
		for _, vc := range vcs {
			if vc != nil {
				if v := vc.Get("pflags"); v != nil {
					if bindPFlags, ok := v.(*pflag.FlagSet); ok {
						if bindPFlags.Lookup(f.Name) != nil {
							envKey = strings.ToUpper(vc.GetString("env_prefix") + "_" + f.Name)
						}
					}
				}
			}
		}

		required := true
		if ann, found := f.Annotations[cobra.BashCompOneRequiredFlag]; !found || (ann[0] != "true") {
			if ann, found = f.Annotations[cobra.BashCompCustom]; !found || (ann[0] != FlagAnnotationCustom) {
				required = false
			}
		}

		deprecated := ""
		if f.Deprecated != "" {
			deprecated = "[deprecated]"
		}
		buf.WriteString(fmt.Sprintln("|", name, "|", envKey, "|", required, "|", f.DefValue, "|", deprecated, f.Usage, "|"))
	}
}

func addDirectoryToZip(zipWriter *zip.Writer, base, uri string, excludes []*regexp.Regexp) error {
	p := path.Join(base, uri)
	entries, err := ioutil.ReadDir(p)
	if err != nil {
		return errors.WithStack(err)
	}
Loop:
	for _, entry := range entries {
		name := entry.Name()
		for _, exclude := range excludes {
			if exclude.MatchString(name) {
				log.Printf("Exclude %s/%s", p, name)
				continue Loop
			}
		}
		if entry.IsDir() {
			err = addDirectoryToZip(zipWriter, base, path.Join(uri, name), excludes)
			if err != nil {
				return err
			}
		} else {
			filePath := path.Join(p, name)
			if entry.Mode()&os.ModeSymlink != 0 {
				filePath, err = os.Readlink(filePath)
				if err != nil {
					return errors.WithStack(err)
				}
				if fi, err := os.Stat(filePath); err != nil {
					return errors.WithStack(err)
				} else {
					if fi.IsDir() {
						err = addDirectoryToZip(zipWriter, base, path.Join(uri, name), excludes)
						if err != nil {
							return err
						}
						continue Loop
					}
				}
			}
			fd, err := os.Open(filePath)
			if err != nil {
				return errors.WithStack(err)
			}

			hdr, err := zip.FileInfoHeader(entry)
			if err != nil {
				fd.Close()
				return errors.WithStack(err)
			}
			hdr.Name = path.Join(uri, name)
			hdr.Method = zip.Deflate
			writer, err := zipWriter.CreateHeader(hdr)
			_, err = io.Copy(writer, fd)
			fd.Close()
		}
	}
	return nil
}

func ZipDirectory(p string, excludes ...string) ([]byte, error) {
	isDir, err := IsDirectory(p)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, errors.New(p + " is not directory")
	}
	regexps := make([]*regexp.Regexp, 0)
	for _, exclude := range excludes {
		re, err := regexp.Compile(exclude)
		if err != nil {
			return nil, err
		}
		regexps = append(regexps, re)
	}

	bs := bytes.NewBuffer(nil)
	zfd := zip.NewWriter(bs)
	if err = addDirectoryToZip(zfd, p, "", regexps); err != nil {
		return nil, err
	}
	if err = zfd.Close(); err != nil {
		return nil, err
	}
	return bs.Bytes(), nil
}

func IsDirectory(p string) (bool, error) {
	f, err := os.Open(p)
	if err != nil {
		return false, err
	}
	fi, err := f.Stat()
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}

var cpuProfileCnt int32 = 0

func startCPUProfile(name string) (*os.File, error) {
	cnt := atomic.AddInt32(&cpuProfileCnt, 1)
	filename := fmt.Sprintf("%s.%03d", name, cnt)
	f, err := os.Create(filename)
	if err != nil {
		return nil, errors.Errorf("fail to create %s for profile err=%+v", filename, err)
	}
	if err = pprof.StartCPUProfile(f); err != nil {
		return nil, errors.Errorf("fail to start profiling err=%+v", err)
	}
	return f, nil
}

func StartCPUProfile(filename string) error {
	if filename != "" {
		fd, err := startCPUProfile(filename)
		if err != nil {
			return err
		}
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGUSR1)
		go func(c chan os.Signal) {
			for {
				<-c
				pprof.StopCPUProfile()
				fd.Close()

				fd, err = startCPUProfile(filename)
				if err != nil {
					log.Panicf("Fail to start CPU Profile err=%+v", err)
				}
			}
		}(c)
	} else {
		return errors.Errorf("filename cannot be empty string")
	}
	return nil
}

func StartMemoryProfile(filename string) error {
	if filename != "" {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGUSR1)
		go func(c chan os.Signal) {
			var memProfileCnt int32 = 0
			for {
				<-c
				cnt := atomic.AddInt32(&memProfileCnt, 1)
				fileName := fmt.Sprintf("%s.%03d", filename, cnt)
				if f, err := os.Create(fileName); err == nil {
					runtime.GC()
					pprof.WriteHeapProfile(f)
					f.Close()
				}
			}
		}(c)
	} else {
		return errors.Errorf("filename cannot be empty string")
	}
	return nil
}

func StartBlockProfile(filename string, rate int) error {
	if filename == "" {
		return errors.Errorf("filename cannot be empty string")
	}
	runtime.SetBlockProfileRate(rate)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)
	go func(c chan os.Signal) {
		var profileCnt int32 = 0
		for {
			<-c
			cnt := atomic.AddInt32(&profileCnt, 1)
			fileName := fmt.Sprintf("%s.%03d", filename, cnt)
			if f, err := os.Create(fileName); err == nil {
				if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
					_ = f.Close()
					_ = os.Remove(fileName)
				} else {
					_ = f.Close()
				}
			}
		}
	}(c)
	return nil
}

func JsonPrettyPrintln(w io.Writer, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Errorf("failed JsonPrettyPrintln v=%+v, err=%+v", v, err)
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func JsonPrettyCopyAndClose(w io.Writer, r io.ReadCloser) error {
	defer r.Close()
	bs, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	buf := bytes.NewBuffer(nil)
	if err := json.Indent(buf, bs, "", "  "); err != nil {
		return err
	}
	_, err = io.Copy(w, buf)
	return err
}

func JsonPrettySaveFile(filename string, perm os.FileMode, v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errors.Errorf("failed JsonPrettySaveFile v=%+v, err=%+v", v, err)
	}
	if err := os.MkdirAll(path.Dir(filename), 0700); err != nil {
		return errors.Errorf("fail to create directory %s err=%+v", filename, err)
	}
	if err := ioutil.WriteFile(filename, b, perm); err != nil {
		return errors.Errorf("fail to save to the file=%s err=%+v", filename, err)
	}
	return err
}

func HttpResponsePrettyPrintln(w io.Writer, resp *http.Response) error {
	if _, err := fmt.Fprintln(w, "Status", resp.Status); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Header", resp.Header); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Response"); err != nil {
		return err
	}
	respB, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("failed read err=%+v", err)
	}
	_, err = fmt.Fprintln(w, string(respB))
	return err
}

var (
	CuiQuitKeyEvtFunc  = func(g *gocui.Gui, v *gocui.View) error { return gocui.ErrQuit }
	CuiQuitUserEvtFunc = func(g *gocui.Gui) error { return gocui.ErrQuit }
	CuiNilUserEvtFunc  = func(g *gocui.Gui) error { return nil }
)

func NewCui() (*gocui.Gui, <-chan bool) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}

	g.SetManagerFunc(func(g *gocui.Gui) error {
		maxX, maxY := g.Size()
		if v, err := g.SetView("main", -1, -1, maxX, maxY); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Wrap = true
			v.Overwrite = true
		}
		return nil
	})

	if err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, CuiQuitKeyEvtFunc); err != nil {
		g.Close()
		log.Panicln(err)
	}
	termCh := make(chan bool)
	go func() {
		defer close(termCh)
		if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
			log.Panicln(err)
		}
		log.Println("gui MainLoop terminate")
	}()
	return g, termCh
}

func TermGui(g *gocui.Gui, termCh <-chan bool) {
	g.Update(CuiQuitUserEvtFunc)
	log.Println("waiting gui terminate")
	select {
	case <-termCh:
		log.Println("gui terminated")
	}
	g.Close()
}

func OnInterrupt(cb func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		cb()
	}()
}
