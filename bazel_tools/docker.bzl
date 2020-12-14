__PUSH_TO_DEV_TMPL = """
#!/bin/sh
set -eE -o functrace
# --- begin runfiles.bash initialization ---
# Copy-pasted from Bazel's Bash runfiles library (tools/bash/runfiles/runfiles.bash).
set -euo pipefail
if [[ ! -d "\$${{RUNFILES_DIR:-/dev/null}}" && ! -f "\$${{RUNFILES_MANIFEST_FILE:-/dev/null}}" ]]; then
    if [[ -f "\$$0.runfiles_manifest" ]]; then
        export RUNFILES_MANIFEST_FILE="\$$0.runfiles_manifest"
    elif [[ -f "\$$0.runfiles/MANIFEST" ]]; then
        export RUNFILES_MANIFEST_FILE="\$$0.runfiles/MANIFEST"
    elif [[ -f "\$$0.runfiles/bazel_tools/tools/bash/runfiles/runfiles.bash" ]]; then
        export RUNFILES_DIR="\$$0.runfiles"
    fi
fi
if [[ -f "\$${{RUNFILES_DIR:-/dev/null}}/bazel_tools/tools/bash/runfiles/runfiles.bash" ]]; then
    source "\$${{RUNFILES_DIR}}/bazel_tools/tools/bash/runfiles/runfiles.bash"
elif [[ -f "\$${{RUNFILES_MANIFEST_FILE:-/dev/null}}" ]]; then
    source "\$$(grep -m1 "^bazel_tools/tools/bash/runfiles/runfiles.bash " \
    	                 "\$$RUNFILES_MANIFEST_FILE" | cut -d ' ' -f 2-)"
else
    echo >&2 "ERROR: cannot find @bazel_tools//tools/bash/runfiles:runfiles.bash"
    exit 1
fi
# --- end runfiles.bash initialization ---
IMAGE="{registry}/{image}:{tag}"
DOCKERFILE=\$$(rlocation __main__/{package}/{dockerfile})
docker build \
      --tag=\$$IMAGE \
      --file=\$$DOCKERFILE \
      \$$(dirname "\$$DOCKERFILE")
docker push \$$IMAGE
"""


def docker_image(name, registry, image, tag, dockerfile, srcs, **kwargs):
    native.genrule(
	name = "%s.sh" % name,
	executable = False,
	outs = ["build_and_push.sh",],
	cmd = """cat > $@ <<EOM
%s
EOM
""" % __PUSH_TO_DEV_TMPL.format(
	    registry = registry,
	    image = image,
	    tag = tag,
	    dockerfile = dockerfile,
	    package = native.package_name(),
	))
    native.sh_binary(
	name = name,
	srcs = ["build_and_push.sh",],
	data = srcs + [dockerfile,],
	deps = ["@bazel_tools//tools/bash/runfiles",])
