//go:build (!darwin && !freebsd) || cgo

package main

func trayOffersDashboardTokenCopy() bool { return true }
