# Makefile for cross-compiling BoringSSL for RISC-V

# RISC-V toolchain prefix
CROSS_COMPILE := riscv64-unknown-linux-gnu-

# Compiler and flags
CC := $(CROSS_COMPILE)gcc
CXX := $(CROSS_COMPILE)g++
AR := $(CROSS_COMPILE)ar

# Check if ranlib is available
RANLIB := $(shell which $(CROSS_COMPILE)ranlib 2>/dev/null)

# Architecture-specific flags
ARCH_FLAGS := -march=rv64gc -mabi=lp64d

# Define OPENSSL_64_BIT for 64-bit RISC-V and OPENSSL_RISCV for RISC-V architecture
DEFINE_FLAGS := -DOPENSSL_64_BIT -DOPENSSL_RISCV

# Add -DOPENSSL_NO_ASM to disable assembly optimizations if needed
CFLAGS := -Wall -O2 $(ARCH_FLAGS) $(DEFINE_FLAGS) -fPIC -DOPENSSL_NO_ASM
CXXFLAGS := $(CFLAGS)

# BoringSSL source directory
BORINGSSL_DIR := src

# Output directory
BUILD_DIR := build_riscv

# Include directories
INCLUDES := -I$(BORINGSSL_DIR)/include

# Source files
CRYPTO_SOURCES := $(wildcard $(BORINGSSL_DIR)/crypto/*.c)
CRYPTO_OBJECTS := $(patsubst $(BORINGSSL_DIR)/crypto/%.c,$(BUILD_DIR)/crypto/%.o,$(CRYPTO_SOURCES))

SSL_SOURCES := $(wildcard $(BORINGSSL_DIR)/ssl/*.c)
SSL_OBJECTS := $(patsubst $(BORINGSSL_DIR)/ssl/%.c,$(BUILD_DIR)/ssl/%.o,$(SSL_SOURCES))

# Targets
all: $(BUILD_DIR) $(BUILD_DIR)/libcrypto.a $(BUILD_DIR)/libssl.a

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

$(BUILD_DIR)/libcrypto.a: $(CRYPTO_OBJECTS)
	$(AR) rcs $@ $^
	$(if $(RANLIB),$(RANLIB) $@)

$(BUILD_DIR)/libssl.a: $(SSL_OBJECTS)
	$(AR) rcs $@ $^
	$(if $(RANLIB),$(RANLIB) $@)

$(BUILD_DIR)/crypto/%.o: $(BORINGSSL_DIR)/crypto/%.c
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $(INCLUDES) -c $< -o $@

$(BUILD_DIR)/ssl/%.o: $(BORINGSSL_DIR)/ssl/%.c
	@mkdir -p $(dir $@)
	$(CC) $(CFLAGS) $(INCLUDES) -c $< -o $@

clean:
	rm -rf $(BUILD_DIR)

.PHONY: all clean
