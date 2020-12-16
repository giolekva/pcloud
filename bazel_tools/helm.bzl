__HELM_INSTALL_TMPL = """
#!/bin/sh
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
CHART_TARBALL=\$$(rlocation __main__/{package}/{chart})
helm --namespace={namespace} install --create-namespace {release_name} \$$CHART_TARBALL {args}
"""

__HELM_UNINSTALL_TMPL = """
helm --namespace={namespace} uninstall {release_name}
"""

def helm_install(name, namespace, release_name, chart, args):
    args_str = ""
    for arg, value in args.items():
        args_str += "--set %s=%s " % (arg, value)
    native.genrule(
        name = "%s.sh" % name,
        executable = False,
        srcs = [chart],
        outs = ["helm_install.sh"],
        cmd = """cat > $@ <<EOM
%s
EOM
""" % __HELM_INSTALL_TMPL.format(
            namespace = namespace,
            release_name = release_name,
            package = native.package_name(),
            chart = "%s.tar.gz" % chart.split(":")[1],
            args = args_str,
        ),
    )
    native.sh_binary(
        name = name,
        srcs = ["helm_install.sh"],
        data = [
            chart,
        ],
        deps = [
            "@bazel_tools//tools/bash/runfiles",
        ],
    )

def helm_uninstall(name, namespace, release_name):
    native.genrule(
        name = "%s.sh" % name,
        executable = False,
        outs = ["helm_uninstall.sh"],
        cmd = """cat > $@ <<EOM
%s
EOM
""" % __HELM_UNINSTALL_TMPL.format(
            namespace = namespace,
            release_name = release_name,
        ),
    )
    native.sh_binary(
        name = name,
        srcs = ["helm_uninstall.sh"],
    )
