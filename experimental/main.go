package main

// // Writer type used to initialize buffer writer
// type Writer int

// func (*Writer) Write(p []byte) (s string) {
// 	return s
// }

func main() {

	// tmpFile, _ := ioutil.TempFile("/tmp", "logfile")

	// defer tmpFile.Close() //nolint
	// defer os.Remove(tmpFile.Name())

	// cmd := exec.Command("echo", "foobaz", "bartap")

	// // get the stdout and stderr stream
	// erc, err := cmd.StderrPipe()
	// if err != nil {
	// 	log.Error().Msg(fmt.Sprintf("Failed to get stderr reader: ", err)
	// }
	// orc, err := cmd.StdoutPipe()
	// if err != nil {
	// 	log.Error().Msg(fmt.Sprintf("Failed to get stdout reader: ", err)
	// }

	// // combine stdout and stderror ReadCloser
	// rc := io.MultiReader(erc, orc)

	// // Prepare the writer
	// f, err := os.OpenFile(tmpFile.Name(), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	// if err != nil {
	// 	logger.Fatalf("Failed to create file")
	// }

	// // cmd.Stderr = os.Stderr
	// // cmd.Stdout = os.Stdout

	// log.Debug().Msg("Executing command: %s", cmd.String())

	// // Command.Start starts a new go routine
	// if err := cmd.Start(); err != nil {
	// 	logger.Fatalf("Failed to start the command: %s", err)
	// }

	// var bufferRead bytes.Buffer
	// teereader := io.TeeReader(rc, &bufferRead)

	// // Everything read from r will be copied to stdout.
	// // a, _ := io.ReadAll(teereader)

	// // b := string(a)

	// log.Debug().Msg("Temp file name: %s", f.Name())

	// if _, err := io.Copy(f, teereader); err != nil {
	// 	logger.Fatalf("Failed to stream to file: %s", err)
	// }

	// if err := cmd.Wait(); err != nil {
	// 	logger.Fatalf("Failed to wait the command to execute: %s", err)
	// }

	// log.Debug().Msg("Buffer: %s", bufferRead.String())

	// // TODO: Should we check the result here?
	// if cmd != nil && cmd.Process != nil {
	// 	cmd.Process.Kill() // nolint
	// }

	// f.Name()

	// var foo = "baz"
	// log.Info().Msg(fmt.Sprintf("I'm here: %s", "manks")
	// fmt.Printf("I'm here: %s", foo)

	Logger_Exp()
}
