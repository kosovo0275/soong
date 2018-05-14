#include <stdio.h>
#include <gtest/gtest.h>
#include <build/version.h>
#include "build_version_test_lib.h"

TEST(BuildNumber, binary) {
  printf("binary version: %s\n", android::build::GetBuildNumber().c_str());
  EXPECT_NE(android::build::GetBuildNumber(), "SOONG BUILD NUMBER PLACEHOLDER");
}

TEST(BuildNumber, library) {
  printf("shared library version: %s\n", LibGetBuildNumber().c_str());
  EXPECT_NE(LibGetBuildNumber(), "SOONG BUILD NUMBER PLACEHOLDER");
}
