package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/common-nighthawk/go-figure"
	"github.com/hashicorp/go-version"
)

var (
	once sync.Once
	info = Info{}
)

// Info contains details about the binary from build time
type Info struct {
	Version   string `json:"version,omitempty"`
	Commit    string `json:"commit,omitempty"`
	Dirty     *bool  `json:"dirty,omitempty"`
	Date      string `json:"date,omitempty"`
	GoVersion string `json:"goVersion,omitempty"`
	Compiler  string `json:"compiler,omitempty"`
	Platform  string `json:"platform,omitempty"`

	Name        string `json:"-"`
	Description string `json:"-"`
}

func getBuildInfo() *debug.BuildInfo {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}

	return bi
}

func getKey(bi *debug.BuildInfo, key string) string {
	if bi == nil {
		return ""
	}

	for _, item := range bi.Settings {
		if item.Key == key {
			return item.Value
		}
	}

	return ""
}

// GetInfo retrieves version information about the binary
func GetInfo() Info {
	once.Do(func() {
		buildInfo := getBuildInfo()
		if buildInfo == nil {
			return
		}

		if len(info.Version) == 0 {
			ver, err := version.NewSemver(buildInfo.Main.Version)
			if err == nil {
				info.Version = ver.Core().String()

				meta := ver.Metadata()
				if len(meta) != 0 {
					info.Version += "+" + meta
				}
			}
		}

		if len(info.Commit) == 0 {
			info.Commit = getKey(buildInfo, "vcs.revision")
		}

		if info.Dirty == nil {
			modified := getKey(buildInfo, "vcs.modified")
			if dirty, err := strconv.ParseBool(modified); err == nil {
				info.Dirty = &dirty
			}
		}

		if len(info.Date) == 0 {
			buildTime := getKey(buildInfo, "vcs.time")
			if t, err := time.Parse("2006-01-02T15:04:05Z", buildTime); err == nil {
				info.Date = t.Format("2006-01-02T15:01:05")
			}
		}

		if len(info.GoVersion) == 0 {
			info.GoVersion = runtime.Version()
		}

		if len(info.Compiler) == 0 {
			info.Compiler = runtime.Compiler
		}

		if len(info.Platform) == 0 {
			info.Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
		}
	})

	return info
}

// String returns the string representation of the version info.
func (i *Info) String() string {
	var b strings.Builder
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)

	if len(i.Name) != 0 {
		fig := figure.NewFigure(strings.ToUpper(i.Name), "basic", true)
		_, _ = fmt.Fprint(w, fig.String())

		_, _ = fmt.Fprint(w, i.Name)
		if len(i.Description) != 0 {
			_, _ = fmt.Fprintf(w, ": %s", i.Description)
		}

		_, _ = fmt.Fprint(w, "\n\n")
	}

	_, _ = fmt.Fprintf(w, "Version:\t%s\n", i.Version)
	_, _ = fmt.Fprintf(w, "Commit:\t%s\n", i.Commit)

	_, _ = fmt.Fprint(w, "Dirty:\t")
	if i.Dirty != nil {
		_, _ = fmt.Fprint(w, strconv.FormatBool(*i.Dirty))
	}
	_, _ = fmt.Fprint(w, "\n")

	_, _ = fmt.Fprintf(w, "Date:\t%s\n", i.Date)
	_, _ = fmt.Fprintf(w, "Go Version:\t%s\n", i.GoVersion)
	_, _ = fmt.Fprintf(w, "Compiler:\t%s\n", i.Compiler)
	_, _ = fmt.Fprintf(w, "Platform:\t%s\n", i.Platform)

	_ = w.Flush()
	return b.String()
}

// JSON returns the JSON representation of the version info.
func (i *Info) JSON() string {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		panic("serializing to json should be infallible")
	}

	return string(b)
}
