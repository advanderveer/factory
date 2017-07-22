package command

//AWSFlags holds options that configure aws
type AWSFlags struct {
	Profile string `long:"aws-profile" description:"AWS Credentials Profile"`
	Region  string `long:"aws-region" description:"AWS Region"`
}

//DebugFlags are used to get more insight into the program behaviour
type DebugFlags struct {
	Debug     bool   `long:"debug" description:"Debug mode enable extra information"`
	Verbosity string `short:"v" long:"verbosity" default:"DEBUG"  description:"Show information at various levels: DEBUG, INFO, WARN, ERROR"`
}
