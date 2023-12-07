package apk

type PackageFilter func(p Package) bool

func AcceptAllPackageFilter() PackageFilter {
	return func(_ Package) bool {
		return true
	}
}
