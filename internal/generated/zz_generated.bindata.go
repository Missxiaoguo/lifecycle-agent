// Code generated for package generated by go-bindata DO NOT EDIT. (@generated)
// sources:
// internal/bindata/prepCleanup.sh
// internal/bindata/prepGetSeedImage.sh
// internal/bindata/prepPullImages.sh
// internal/bindata/prepSetupStateroot.sh
package generated

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _prepcleanupSh = []byte(`#!/bin/bash
#
# Image Based Upgrade Prep Stage Cleanup
#

declare SEED_IMAGE=

function usage {
    cat <<ENDUSAGE
Parameters:
    --seed-image <image>
ENDUSAGE
    exit 1
}

function cleanup {
    if [ -n "${SEED_IMAGE}" ]; then
        if podman image exists "${SEED_IMAGE}"; then
            podman image unmount "${SEED_IMAGE}"
            podman rmi "${SEED_IMAGE}"
        fi
    fi
}

LONGOPTS="seed-image:"
OPTS=$(getopt -o h --long "${LONGOPTS}" --name "$0" -- "$@")

eval set -- "${OPTS}"

while :; do
    case "$1" in
        --seed-image)
            SEED_IMAGE=$2
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

cleanup

`)

func prepcleanupShBytes() ([]byte, error) {
	return _prepcleanupSh, nil
}

func prepcleanupSh() (*asset, error) {
	bytes, err := prepcleanupShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "prepCleanup.sh", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _prepgetseedimageSh = []byte(`#!/bin/bash
#
# Image Based Upgrade Prep - Pull seed image
#

declare SEED_IMAGE=
declare PROGRESS_FILE=

function usage {
    cat <<ENDUSAGE
Parameters:
    --seed-image <image>
    --progress-file <file>
ENDUSAGE
    exit 1
}

function set_progress {
    echo "$1" > "${PROGRESS_FILE}"
}

function fatal {
    set_progress "Failed"
    echo "$@" >&2
    exit 1
}

function log_it {
    echo "$@" | tr '[:print:]' -
    echo "$@"
    echo "$@" | tr '[:print:]' -
}

LONGOPTS="seed-image:,progress-file:"
OPTS=$(getopt -o h --long "${LONGOPTS}" --name "$0" -- "$@")

eval set -- "${OPTS}"

while :; do
    case "$1" in
        --seed-image)
            SEED_IMAGE=$2
            shift 2
            ;;
        --progress-file)
            PROGRESS_FILE=$2
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

set_progress "started-seed-image-pull"

log_it "Pulling and mounting seed image"
podman pull "${SEED_IMAGE}" || fatal "Failed to pull image: ${SEED_IMAGE}"

img_mnt=$(podman image mount "${SEED_IMAGE}")
if [ -z "${img_mnt}" ]; then
    fatal "Failed to mount image: ${SEED_IMAGE}"
fi

# Collect / validate information? Verify required files exist?

set_progress "completed-seed-image-pull"

log_it "Pulled seed image"

exit 0
`)

func prepgetseedimageShBytes() ([]byte, error) {
	return _prepgetseedimageSh, nil
}

func prepgetseedimageSh() (*asset, error) {
	bytes, err := prepgetseedimageShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "prepGetSeedImage.sh", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _preppullimagesSh = []byte(`#!/bin/bash
#
# Image Based Upgrade Prep - Precache images
#

declare SEED_IMAGE=
declare PROGRESS_FILE=

function usage {
    cat <<ENDUSAGE
Parameters:
    --seed-image <image>
    --progress-file <file>
ENDUSAGE
    exit 1
}

function set_progress {
    echo "$1" > "${PROGRESS_FILE}"
}

function fatal {
    set_progress "Failed"
    echo "$@" >&2
    exit 1
}

function log_it {
    echo "$@" | tr '[:print:]' -
    echo "$@"
    echo "$@" | tr '[:print:]' -
}

function build_catalog_regex {
    if grep -q . "${img_mnt}/catalogimages.list"; then
        awk -F: '{print $1 ":"; print $1 "@";}' "${img_mnt}/catalogimages.list" | paste -sd\|
    fi
}

LONGOPTS="seed-image:,progress-file:"
OPTS=$(getopt -o h --long "${LONGOPTS}" --name "$0" -- "$@")

eval set -- "${OPTS}"

while :; do
    case "$1" in
        --seed-image)
            SEED_IMAGE=$2
            shift 2
            ;;
        --progress-file)
            PROGRESS_FILE=$2
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

set_progress "started-precache"

# Image should already be pulled and mounted
img_mnt=$(podman image mount --format json | jq -r --arg img "${SEED_IMAGE}" '.[] | select(.Repositories[0] == $img) | .mountpoint')
if [ -z "${img_mnt}" ]; then
    fatal "Seed image is not mounted: ${SEED_IMAGE}"
fi

log_it "Precaching non-catalog images"
grep -vE "$(build_catalog_regex)" "${img_mnt}/containers.list" | xargs --no-run-if-empty --max-args 1 --max-procs 10 crictl pull

log_it "Precaching catalog images"
if grep -q . "${img_mnt}/catalogimages.list"; then
    xargs --no-run-if-empty --max-args 1 --max-procs 10 crictl pull < "${img_mnt}/catalogimages.list"
fi

set_progress "completed-precache"

exit 0
`)

func preppullimagesShBytes() ([]byte, error) {
	return _preppullimagesSh, nil
}

func preppullimagesSh() (*asset, error) {
	bytes, err := preppullimagesShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "prepPullImages.sh", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _prepsetupstaterootSh = []byte(`#!/bin/bash
#
# Image Based Upgrade Prep - Setup new stateroot
#

declare SEED_IMAGE=
declare PROGRESS_FILE=
declare WORKDIR=

function usage {
    cat <<ENDUSAGE
Parameters:
    --seed-image <image>
    --progress-file <file>
ENDUSAGE
    exit 1
}

function set_progress {
    echo "$1" > "${PROGRESS_FILE}"
}

function fatal {
    set_progress "Failed"
    echo "$@" >&2
    exit 1
}

function cleanup {
    if [ -n "${WORKDIR}" ]; then
        rm -rf "${WORKDIR}"
    fi
}

trap cleanup EXIT

function log_it {
    echo "$@" | tr '[:print:]' -
    echo "$@"
    echo "$@" | tr '[:print:]' -
}

function build_kargs {
    jq -r '.spec.kernelArguments[]' "${img_mnt}/mco-currentconfig.json" \
        | xargs --no-run-if-empty -I% echo -n "--karg-append % "
}

LONGOPTS="seed-image:,progress-file:"
OPTS=$(getopt -o h --long "${LONGOPTS}" --name "$0" -- "$@")

eval set -- "${OPTS}"

while :; do
    case "$1" in
        --seed-image)
            SEED_IMAGE=$2
            shift 2
            ;;
        --progress-file)
            PROGRESS_FILE=$2
            shift 2
            ;;
        --os-version)
            CR_OS_VERSION=$2
            shift 2
            ;;
        --os-name)
            CR_OS_NAME=$2
            shift 2
            ;;
        --)
            shift
            break
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

set_progress "started-stateroot"

WORKDIR=$(mktemp -d -p /var/tmp)
if [ -z "${WORKDIR}" ]; then
    fatal "Failed to create workdir"
fi

export KUBECONFIG=/etc/kubernetes/static-pod-resources/kube-apiserver-certs/secrets/node-kubeconfigs/lb-ext.kubeconfig

mount /sysroot -o remount,rw || fatal "Failed to remount /sysroot"

# Image should already be pulled and mounted
img_mnt=$(podman image mount --format json | jq -r --arg img "${SEED_IMAGE}" '.[] | select(.Repositories[0] == $img) | .mountpoint')
if [ -z "${img_mnt}" ]; then
    fatal "Seed image is not mounted: ${SEED_IMAGE}"
fi

ostree_repo="${WORKDIR}/ostree"
mkdir -p "${ostree_repo}" || fatal "Failed to create dir: ${ostree_repo}"

tar xzf "${img_mnt}/ostree.tgz" --selinux -C "${ostree_repo}"

# Collect seed deployment data from the backup
upg_booted_id=$(jq -r '.deployments[] | select(.booted == true) | .id' "${img_mnt}/rpm-ostree.json")
upg_booted_deployment=${upg_booted_id/*-}
upg_booted_ref=${upg_booted_deployment/\.*}

if [ -z "${upg_booted_id}" ] || [ -z "${upg_booted_deployment}" ] || [ -z "${upg_booted_ref}" ]; then
    fatal "Failed to identify deployment from seed image"
fi

up_ver=$(jq -r '.status.desired.version' "${img_mnt}/clusterversion.json")
if [ -z "${up_ver}" ]; then
    fatal "Failed to identify version from seed image"
fi

if [ ${CR_OS_VERSION} -ne ${up_ver} ]; then
    fatal "Seed image version(${up_ver}) differs from version specified in ibu.Spec.SeedImageRef.Version(${CR_OS_VERSION})"
fi

new_osname="${CR_OS_NAME}"

log_it "Importing remote ostree"
ostree pull-local "${ostree_repo}" || fatal "Failed: ostree pull-local ${ostree_repo}"
ostree admin os-init "${new_osname}" || fatal "Failed: ostree admin os-init ${new_osname}"

log_it "Creating new deployment ${new_osname}"
# We should create the new deploy as not-default, and after the whole process is done, be able to switch to it for the next reboot
# ostree admin deploy --os ${new_osname} $(build_kargs) --not-as-default ${upg_booted_ref}
# Until I find how to do the switch, I'll deploy as default
# shellcheck disable=SC2046
ostree admin deploy --os "${new_osname}" $(build_kargs) "${upg_booted_ref}" || fatal "Failed ostree admin deploy"
ostree_deploy=$(ostree admin status | awk /"${new_osname}"/'{print $2}')
if [ -z "${ostree_deploy}" ]; then
    fatal "Unable to determine deployment"
fi

# Restore the seed .origin file
cp "${img_mnt}/ostree-${upg_booted_deployment}.origin" "/ostree/deploy/${new_osname}/deploy/${ostree_deploy}.origin" || fatal "Failed to copy .origin from seed"

log_it "Restoring /var"
tar xzf "${img_mnt}/var.tgz" -C "/ostree/deploy/${new_osname}" --selinux || fatal "Failed to restore /var"

log_it "Restoring /etc"
tar xzf "${img_mnt}/etc.tgz" -C "/ostree/deploy/${new_osname}/deploy/${ostree_deploy}" --selinux || fatal "Failed to restore /etc"
xargs --no-run-if-empty -ifile rm -f /ostree/deploy/"${new_osname}"/deploy/"${ostree_deploy}"/file < "${img_mnt}/etc.deletions"

log_it "Waiting for API"
until oc get clusterversion &>/dev/null; do
    sleep 5
done

log_it "Backing up certificates to be used by recert"
certs_dir="/ostree/deploy/${new_osname}/var/opt/openshift/certs"
mkdir -p "${certs_dir}"
oc extract -n openshift-config configmap/admin-kubeconfig-client-ca --keys=ca-bundle.crt --to=- > "${certs_dir}/admin-kubeconfig-client-ca.crt" \
    || fatal "Failed: oc extract -n openshift-config configmap/admin-kubeconfig-client-ca --keys=ca-bundle.crt"
for key in {loadbalancer,localhost,service-network}-serving-signer; do
    oc extract -n openshift-kube-apiserver-operator secret/${key} --keys=tls.key --to=- > "${certs_dir}/${key}.key" \
        || fatal "Failed: oc extract -n openshift-kube-apiserver-operator secret/${key} --keys=tls.key"
done
ingress_cn=$(oc extract -n openshift-ingress-operator secret/router-ca --keys=tls.crt --to=- | openssl x509 -subject -noout -nameopt multiline | awk '/commonName/{print $3}')
if [ -z "${ingress_cn}" ]; then
    fatal "Failed to get ingress_cn"
fi
oc extract -n openshift-ingress-operator secret/router-ca --keys=tls.key --to=- > "${certs_dir}/ingresskey-${ingress_cn}" \
    || fatal "Failed: oc extract -n openshift-ingress-operator secret/router-ca --keys=tls.key"

set_progress "completed-stateroot"

exit 0
`)

func prepsetupstaterootShBytes() ([]byte, error) {
	return _prepsetupstaterootSh, nil
}

func prepsetupstaterootSh() (*asset, error) {
	bytes, err := prepsetupstaterootShBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "prepSetupStateroot.sh", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"prepCleanup.sh":        prepcleanupSh,
	"prepGetSeedImage.sh":   prepgetseedimageSh,
	"prepPullImages.sh":     preppullimagesSh,
	"prepSetupStateroot.sh": prepsetupstaterootSh,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//
//	data/
//	  foo.txt
//	  img/
//	    a.png
//	    b.png
//
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"prepCleanup.sh":        {prepcleanupSh, map[string]*bintree{}},
	"prepGetSeedImage.sh":   {prepgetseedimageSh, map[string]*bintree{}},
	"prepPullImages.sh":     {preppullimagesSh, map[string]*bintree{}},
	"prepSetupStateroot.sh": {prepsetupstaterootSh, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
