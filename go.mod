module github.com/jmontleon/crane-test

go 1.16

require (
	github.com/konveyor/crane-lib v0.0.0-20210701174456-0792fb1929de
	github.com/openshift/api v0.0.0-20210625082935-ad54d363d274
	k8s.io/api v0.21.2
	k8s.io/apimachinery v0.21.2
	k8s.io/client-go v0.21.2
	sigs.k8s.io/controller-runtime v0.9.2
)

replace github.com/konveyor/crane-lib => /home/jason/Documents/openshift/src/github.com/konveyor/crane-lib

replace bitbucket.org/ww/goautoneg v0.0.0-20120707110453-75cd24fc2f2c => github.com/markusthoemmes/goautoneg v0.0.0-20190713162725-c6008fefa5b

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20190716152234-9ea19f9dd578
