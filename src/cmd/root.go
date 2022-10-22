package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const EnvDebugMode = "PC_DEBUG_MODE"

var (
	fileName string
	port     int
	isTui    bool
	version  string

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "process-compose",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(fileName)
			if !cmd.Flags().Changed("config") {

				pwd, err := os.Getwd()
				if err != nil {
					log.Fatal().Msg(err.Error())
				}
				file, err := app.AutoDiscoverComposeFile(pwd)
				if err != nil {
					log.Fatal().Msg(err.Error())
				}
				fileName = file
			}

			if os.Getenv(EnvDebugMode) == "" {
				gin.SetMode(gin.ReleaseMode)
			}

			routersInit := api.InitRoutes(!isTui)
			readTimeout := time.Duration(60) * time.Second
			writeTimeout := time.Duration(60) * time.Second
			endPoint := fmt.Sprintf(":%d", port)
			maxHeaderBytes := 1 << 20

			server := &http.Server{
				Addr:           endPoint,
				Handler:        routersInit,
				ReadTimeout:    readTimeout,
				WriteTimeout:   writeTimeout,
				MaxHeaderBytes: maxHeaderBytes,
			}

			log.Info().Msgf("start http server listening %s", endPoint)

			go server.ListenAndServe()

			project := app.CreateProject(fileName)

			if isTui {
				defer quiet()()
				go project.Run()
				tui.SetupTui(version, project.LogLength)
			} else {
				runHeadless(project)
			}

			log.Info().Msg("Thank you for using proccess-compose")
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ver string) {
	version = ver
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	rootCmd.Flags().StringVarP(&fileName, "config", "f", app.DefaultFileNames[0], "path to config file to load")
	rootCmd.Flags().BoolVarP(&isTui, "tui", "t", true, "disable tui (-t=false)")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "port number")
}

func runHeadless(project *app.Project) {
	cancelChan := make(chan os.Signal, 1)
	// catch SIGTERM or SIGINTERRUPT
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	go project.Run()
	sig := <-cancelChan
	log.Info().Msgf("Caught %v - Shutting down the running processes...", sig)
	project.ShutDownProject()
}

func quiet() func() {
	null, _ := os.Open(os.DevNull)
	sout := os.Stdout
	serr := os.Stderr
	os.Stdout = null
	os.Stderr = null
	zerolog.SetGlobalLevel(zerolog.Disabled)
	return func() {
		defer null.Close()
		os.Stdout = sout
		os.Stderr = serr
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}