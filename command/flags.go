package command

//AWSFlags holds options that configure aws
type AWSFlags struct {
	Profile string `long:"aws-profile" description:"AWS Credentials Profile"`
}

//DebugFlags are used to get more insight into the program behaviour
type DebugFlags struct {
	Debug   bool   `long:"debug" description:"Debug mode enable extra information"`
	Verbose []bool `short:"v" long:"verbose" description:"Show information at various levels"`
}
