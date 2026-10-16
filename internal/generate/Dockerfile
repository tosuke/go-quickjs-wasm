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
COPY internal/go.mod internal/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

FROM tools AS generate
ARG QUICKJS_VERSION=2026-06-04

RUN mkdir -p .build \
    && curl -fL "https://bellard.org/quickjs/quickjs-${QUICKJS_VERSION}.tar.xz" \
        -o .build/quickjs.tar.xz \
    && echo 'b376e839b322978313d929fd20663b11ba58b75df5a46c126dd19ea2fa70ad2a  .build/quickjs.tar.xz' | sha256sum -c - \
    && tar -xf .build/quickjs.tar.xz -C .build

# Disable OS threads, keep stack checks, and declare malloc_usable_size.
COPY internal/generate/quickjs.patch ./generate/
RUN patch --batch --fuzz=0 -d ".build/quickjs-${QUICKJS_VERSION}" -p1 < generate/quickjs.patch

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go tool libc-gen -c-out .build/libc

COPY internal/generate/shim.c internal/generate/freestanding.c internal/generate/exports.txt ./generate/
RUN awk '{print "--export=" $0}' generate/exports.txt > .build/exports.txt
# Compile and link separately: Clang's wasm link driver invokes wasm-opt without
# -g, which discards names/DWARF before our explicitly configured optimizer runs.
RUN src=".build/quickjs-${QUICKJS_VERSION}" \
    && clang --target=wasm32-unknown-unknown -ffreestanding -nostdlib -mbulk-memory \
        -Os -g -c -fno-strict-aliasing -ffunction-sections -fdata-sections \
        -I.build/libc -I"$src" -DCONFIG_VERSION=\"${QUICKJS_VERSION}\" \
        "$src/quickjs.c" "$src/dtoa.c" "$src/libregexp.c" \
        "$src/libunicode.c" "$src/cutils.c" \
        generate/shim.c generate/freestanding.c \
    && wasm-ld --no-entry --allow-undefined --export-memory \
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
        --enable-wide-arithmetic \
        --strip-producers --strip-dwarf

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out \
    && go tool libc-gen -wasm .build/quickjs.wasm -pkg quickjs_wasm -o /out/libc_generated.go \
    && go tool wasm2go -embed -unsafe -provided /out/libc_generated.go -pkg quickjs_wasm \
        -o /out/quickjs_generated.go .build/quickjs.wasm

# Export only generated files.
FROM scratch AS artifacts
COPY --from=generate /out/ /
