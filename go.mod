module github.com/jmontleon/crane-test

go 1.16

require (
	github.com/jmontleon/crane-lib v0.0.0-20210506213753-104d5f8d1092 // indirect
	github.com/openshift/api v0.0.0-20210503193030-25175d9d392d // indirect
	k8s.io/api v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)

replace bitbucket.org/ww/goautoneg v0.0.0-20120707110453-75cd24fc2f2c => github.com/markusthoemmes/goautoneg v0.0.0-20190713162725-c6008fefa5b

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20190716152234-9ea19f9dd578
