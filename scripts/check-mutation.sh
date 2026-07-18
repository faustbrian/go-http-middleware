#!/usr/bin/env bash
set -euo pipefail

workspace=$(mktemp -d)
trap 'rm -rf "$workspace"' EXIT
baseline="$workspace/baseline"
mkdir -p "$baseline"
tar --exclude=.git --exclude=coverage.out -cf - . | tar -xf - -C "$baseline"

run_mutant() {
	local name=$1 file=$2 from=$3 to=$4 package=$5
	local mutant="$workspace/$name"
	mkdir -p "$mutant"
	tar -cf - -C "$baseline" . | tar -xf - -C "$mutant"
	FROM="$from" TO="$to" perl -0pi -e '
$changed = s/\Q$ENV{FROM}\E/$ENV{TO}/;
END { die "mutation source not found: $ENV{FROM}\n" unless $changed }
' "$mutant/$file"
	if (cd "$mutant" && go test "$package" >mutation.log 2>&1); then
		echo "survived mutation: $name" >&2
		cat "$mutant/mutation.log" >&2
		exit 1
	fi
	printf 'killed mutation: %s\n' "$name"
	rm -rf "$mutant"
}

run_mutant chain_depth chain.go 'len(middleware) > MaxChainDepth' 'len(middleware) < MaxChainDepth' .
run_mutant duplicate chain.go '!previous.allowDuplicate || !descriptor.allowDuplicate' '!previous.allowDuplicate && !descriptor.allowDuplicate' .
run_mutant condition chain.go 'if predicate(r) {' 'if !predicate(r) {' .
run_mutant request_id_trust requestid/requestid.go 'policy.TrustInbound && len(values) == 1' '!policy.TrustInbound && len(values) == 1' ./requestid
run_mutant body_known_limit bodylimit/bodylimit.go 'r.ContentLength > policy.MaxBytes' 'r.ContentLength < policy.MaxBytes' ./bodylimit
run_mutant timeout_bound deadline/timeout.go 'w.payload.Len()+len(payload) > w.maximum' 'w.payload.Len()+len(payload) < w.maximum' ./deadline
run_mutant proxy_trust proxy/proxy.go 'if isTrusted(info.ClientIP, trusted) {' 'if !isTrusted(info.ClientIP, trusted) {' ./proxy
run_mutant proxy_client proxy/proxy.go 'index >= 0 && isTrusted(current, trusted)' 'index >= 0 && !isTrusted(current, trusted)' ./proxy
run_mutant cors_origin cors/cors.go 'if configuration.origins[origin] {' 'if !configuration.origins[origin] {' ./cors
run_mutant cors_method cors/cors.go '!c.methodsWildcard && !c.methods[method]' '!c.methodsWildcard && c.methods[method]' ./cors
run_mutant hsts_ack secureheader/secureheader.go 'policy.HSTS != "" && !policy.AcknowledgeHSTS' 'policy.HSTS != "" && policy.AcknowledgeHSTS' ./secureheader
run_mutant coding_quality compress/compress.go 'gzipQ <= 0 || gzipQ < identityQ' 'gzipQ <= 0 && gzipQ < identityQ' ./compress
run_mutant compression_size compress/compress.go 'w.buffer.Len() < minimum' 'w.buffer.Len() > minimum' ./compress
run_mutant accept_quality content/content.go 'q > 0 && matchesAny' 'q < 0 && matchesAny' ./content
run_mutant admission_limit admission/admission.go 'policy.MaxInFlight < 1' 'policy.MaxInFlight > 1' ./admission
run_mutant no_store responsepolicy/responsepolicy.go 'header.Set("Cache-Control", "no-store")' 'header.Set("Cache-Control", "public")' ./responsepolicy
run_mutant panic_commit recovery/recovery.go 'if !recorder.Committed {' 'if recorder.Committed {' ./recovery
run_mutant observer_panic observe/observe.go 'if panicValue != nil {' 'if panicValue == nil {' ./observe

echo 'mutation score: 18/18 killed (100.0%)'
