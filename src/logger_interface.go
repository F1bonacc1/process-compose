package main

type PcLogger interface {
	Info(message string, process string, replica int)
	Error(message string, process string, replica int)
	Close()
}
