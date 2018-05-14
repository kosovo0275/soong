#include <build/version.h>
#include "build_version_test_lib.h"

std::string LibGetBuildNumber() {
  return android::build::GetBuildNumber();
}
