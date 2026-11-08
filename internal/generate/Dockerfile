# syntax=docker/dockerfile:1
FROM golang:1.26.2-trixie AS tools

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        clang-19 lld-19 patch curl ca-certificates xz-utils \
    && rm -rf /var/lib/apt/lists/*

# Pin the optimizer independently of the distribution's Binaryen package.
RUN curl -fL \
        https://github.com/WebAssembly/binaryen/releases/download/version_133/binaryen-version_133-x86_64-linux.tar.gz \
        -o /tmp/binaryen.tar.gz \
    && echo '2dc9c7813f5375db93d96ead4b78222fcc3e2677bbb832297af4797782a37489  /tmp/binaryen.tar.gz' | sha256sum -c - \
    && tar -xf /tmp/binaryen.tar.gz -C /opt \
    && rm /tmp/binaryen.tar.gz

ENV PATH="/usr/lib/llvm-19/bin:/opt/binaryen-version_133/bin:$PATH" \
    GOTOOLCHAIN=local

WORKDIR /src/internal
RUN --mount=type=bind,source=internal/go.mod,target=go.mod \
    --mount=type=bind,source=internal/go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM tools AS generate-base
ARG QUICKJS_VERSION=2026-06-04

RUN mkdir -p .build /out \
    && curl -fL "https://bellard.org/quickjs/quickjs-${QUICKJS_VERSION}.tar.xz" \
        -o .build/quickjs.tar.xz \
    && echo 'b376e839b322978313d929fd20663b11ba58b75df5a46c126dd19ea2fa70ad2a  .build/quickjs.tar.xz' | sha256sum -c - \
    && tar -xf .build/quickjs.tar.xz -C .build

# Use Wasm atomics instead of pthreads, keep stack checks, and declare malloc_usable_size.
RUN --mount=type=bind,source=internal/generate/quickjs.patch,target=/tmp/quickjs.patch \
    patch --batch --fuzz=0 -d ".build/quickjs-${QUICKJS_VERSION}" -p1 < /tmp/quickjs.patch

RUN --mount=type=bind,source=internal/go.mod,target=go.mod \
    --mount=type=bind,source=internal/go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go tool libc-gen -c-out .build/libc

RUN --mount=type=bind,source=internal/generate/libc.patch,target=/tmp/libc.patch \
    patch --batch --fuzz=0 -d .build/libc -p1 < /tmp/libc.patch

FROM generate-base AS generate
ARG MEMORY_INITIAL_PAGES=64
ARG MEMORY_MAX_PAGES=65536

# Compile and link separately: Clang's wasm link driver invokes wasm-opt without
# -g, which discards names/DWARF before our explicitly configured optimizer runs.
RUN --mount=type=bind,source=internal/generate/shim.c,target=generate/shim.c \
    --mount=type=bind,source=internal/generate/freestanding.c,target=generate/freestanding.c \
    --mount=type=bind,source=internal/generate/pthread.h,target=generate/pthread.h \
    src=".build/quickjs-${QUICKJS_VERSION}" \
    && clang --target=wasm32-unknown-unknown -ffreestanding -nostdlib \
        -mmutable-globals -mbulk-memory -mreference-types \
        -mnontrapping-fptoint -msign-ext -mextended-const -mtail-call -pthread \
        -mmultivalue -Xclang -target-abi -Xclang experimental-mv \
        -Os -g -c -fno-strict-aliasing -ffunction-sections -fdata-sections \
        -Igenerate -I.build/libc -I"$src" -DCONFIG_VERSION=\"${QUICKJS_VERSION}\" \
        "$src/quickjs.c" "$src/dtoa.c" "$src/libregexp.c" \
        "$src/libunicode.c" "$src/cutils.c" \
        generate/shim.c generate/freestanding.c

RUN --mount=type=bind,source=internal/generate/exports.txt,target=generate/exports.txt \
    awk '{print "--export=" $0}' generate/exports.txt > .build/exports.txt

RUN wasm-ld --no-entry --allow-undefined --import-memory --export-memory \
        --shared-memory --initial-memory=$((MEMORY_INITIAL_PAGES * 65536)) \
        --max-memory=$((MEMORY_MAX_PAGES * 65536)) \
        --export=__stack_pointer --export=__tls_base --export=__tls_size \
        --export=__tls_align --export=__wasm_init_tls \
        -z stack-size=2097152 --export-table @.build/exports.txt \
        quickjs.o dtoa.o libregexp.o libunicode.o cutils.o shim.o freestanding.o \
        -o .build/quickjs-unoptimized.wasm

RUN wasm-opt -g .build/quickjs-unoptimized.wasm -o .build/quickjs.wasm \
        --converge \
        --gufa-optimizing --generate-global-effects \
        --low-memory-unused \
        --flatten --rereloop --const-hoisting -O4 \
        --enable-mutable-globals --enable-multivalue \
        --enable-nontrapping-float-to-int --enable-sign-ext \
        --enable-reference-types --enable-bulk-memory \
        --enable-extended-const --enable-tail-call \
        --enable-wide-arithmetic --enable-threads \
        --strip-producers --strip-dwarf

RUN --mount=type=bind,source=internal/go.mod,target=go.mod \
    --mount=type=bind,source=internal/go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go tool libc-gen -deref-mem -wasm .build/quickjs.wasm -pkg quickjs_wasm -o /out/libc_generated.go

RUN --mount=type=bind,source=internal/generate/modulegen,target=generate/modulegen \
    --mount=type=bind,source=internal/go.mod,target=go.mod \
    --mount=type=bind,source=internal/go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go tool wasm2go -embed -unsafe -provided /out/libc_generated.go -pkg quickjs_wasm \
        -o /out/quickjs_generated.go .build/quickjs.wasm \
    && go run ./generate/modulegen -file /out/quickjs_generated.go -libc /out/libc_generated.go

# Keep New() as the public constructor; its imported memory uses these limits.
RUN printf '%s\n' \
        '// Code generated by internal/generate/Dockerfile; DO NOT EDIT.' \
        '' \
        'package quickjs_wasm' \
        '' \
        "const memoryInitialPages = ${MEMORY_INITIAL_PAGES}" \
        "const memoryMaxPages = ${MEMORY_MAX_PAGES}" \
        > /out/memory_limits_generated.go

RUN printf '%s\n' \
        '// Code generated by internal/generate/Dockerfile; DO NOT EDIT.' \
        '' \
        'package quickjs_wasm' \
        '' \
        "const Version = \"${QUICKJS_VERSION}\"" \
        > /out/version_generated.go

FROM generate-base AS generate-test

# Link the test shim as a side module that imports the main module's state.
RUN --mount=type=bind,source=internal/generate/shim_test.c,target=generate/shim_test.c \
    src=".build/quickjs-${QUICKJS_VERSION}" \
    && clang --target=wasm32-unknown-unknown -ffreestanding -nostdlib \
        -Os -fPIC -matomics -mbulk-memory -fno-strict-aliasing -I.build/libc -I"$src" \
        -c generate/shim_test.c -o .build/test_shim.o

RUN wasm-ld --shared --shared-memory --max-memory=4294967296 \
        --no-entry --import-undefined .build/test_shim.o -o .build/test_shim.wasm

RUN --mount=type=bind,source=internal/generate/modulegen,target=generate/modulegen \
    --mount=type=bind,source=internal/go.mod,target=go.mod \
    --mount=type=bind,source=internal/go.sum,target=go.sum \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go tool wasm2go -unsafe -pkg quickjs_wasm_test \
        -o /out/quickjs_generated_test.go .build/test_shim.wasm \
    && go run ./generate/modulegen -file /out/quickjs_generated_test.go -constructor New

FROM generate-base AS generate-structs

# Read the API schema in Go and compile layout probes for the actual C target.
RUN --mount=type=bind,source=internal/go.mod,target=go.mod \
    --mount=type=bind,source=internal/go.sum,target=go.sum \
    --mount=type=bind,source=quickjs.go,target=/src/quickjs.go \
    --mount=type=bind,source=internal/generate/structgen,target=generate/structgen \
    --mount=type=cache,target=/root/.cache/go-build \
    go run ./generate/structgen -I .build/libc -I ".build/quickjs-${QUICKJS_VERSION}"

# Export only generated files.
FROM scratch AS artifacts
COPY --link --from=generate /out/ /
COPY --link --from=generate-test /out/ /
COPY --link --from=generate-structs /out/ /
