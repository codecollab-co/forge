package templates

// License bodies. Apache and GPL use the canonical "see <URL>" short form
// rather than the full multi-page text — what most repos actually do for
// their LICENSE file in practice. Users who need the full text can paste
// it themselves; we'd rather a small repo than an exhaustive copy.

const licenseMIT = `MIT License

Copyright (c) [year] [fullname]

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`

const licenseApache20 = `                                 Apache License
                           Version 2.0, January 2004
                        http://www.apache.org/licenses/

Copyright [year] [fullname]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
`

const licenseBSD3 = `BSD 3-Clause License

Copyright (c) [year], [fullname]
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
   may be used to endorse or promote products derived from this software
   without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
`

const licenseGPL3 = `Copyright (C) [year] [fullname]

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

Full text: https://www.gnu.org/licenses/gpl-3.0.en.html
`

const licenseISC = `ISC License

Copyright (c) [year] [fullname]

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR
OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
PERFORMANCE OF THIS SOFTWARE.
`

const licenseUnlicense = `This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or distribute
this software, either in source code form or as a compiled binary, for any
purpose, commercial or non-commercial, and by any means.

In jurisdictions that recognize copyright laws, the author or authors of this
software dedicate any and all copyright interest in the software to the public
domain. We make this dedication for the benefit of the public at large and to
the detriment of our heirs and successors.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.

For more information, please refer to <http://unlicense.org/>
`

const gitignorePython = `# Byte-compiled / optimized / DLL files
__pycache__/
*.py[cod]
*$py.class

# Distribution / packaging
build/
dist/
*.egg-info/
*.egg

# Virtual environments
.venv/
venv/
env/

# Testing / coverage
.tox/
.coverage
htmlcov/
.pytest_cache/
.mypy_cache/
.ruff_cache/

# Editors
.idea/
.vscode/
.DS_Store

# Local env
.env
`

const gitignoreNode = `node_modules/
npm-debug.log*
yarn-debug.log*
yarn-error.log*
pnpm-debug.log*

# Build output
dist/
build/
.next/
.nuxt/
.cache/

# Coverage
coverage/

# Editors
.idea/
.vscode/
.DS_Store

# Env
.env
.env.local
`

const gitignoreGo = `# Binaries
bin/
*.exe
*.test
*.out

# Vendor
vendor/

# Coverage
*.cover
coverage.txt

# Editors
.idea/
.vscode/
.DS_Store

# Env
.env
`

const gitignoreRust = `target/
Cargo.lock
**/*.rs.bk
*.pdb

# Editors
.idea/
.vscode/
.DS_Store
`

const gitignoreJava = `# Compiled class files
*.class

# Build outputs
target/
build/
out/
*.jar
*.war
*.ear

# Maven / Gradle
.m2/
.gradle/

# Editors
.idea/
*.iml
.vscode/
.DS_Store
`

const gitignoreKotlin = `# Compiled
*.class

# Build
build/
out/
.gradle/

# IDE
.idea/
*.iml
.vscode/
.DS_Store

# Kotlin
*.kotlin_module
`

const gitignoreC = `# Compiled
*.o
*.obj
*.so
*.a
*.exe

# Build
build/
cmake-build-*/

# Editors
.idea/
.vscode/
.DS_Store
`

const gitignoreCPP = `# Compiled
*.o
*.obj
*.so
*.a
*.exe
*.lo
*.la

# Build
build/
cmake-build-*/

# Editors
.idea/
.vscode/
.DS_Store
`

const gitignoreRuby = `*.gem
*.rbc
.bundle/
.config
coverage/
InstalledFiles
log/
pkg/
spec/reports/
spec/examples.txt
test/tmp/
test/version_tmp/
tmp/
.rspec
.rvmrc
.ruby-version
vendor/

# Editors
.idea/
.vscode/
.DS_Store
`

const gitignoreSwift = `# Xcode
build/
DerivedData/
*.pbxuser
*.mode1v3
*.mode2v3
*.perspectivev3
xcuserdata/

# Swift Package Manager
.swiftpm/
.build/
Packages/

# CocoaPods
Pods/

# Editors
.idea/
.vscode/
.DS_Store
`

const gitignoreAndroid = `# Built artifacts
build/
*.apk
*.aab
*.dex
*.class

# Gradle
.gradle/
local.properties

# Android Studio
.idea/
*.iml
captures/

# Keystore
*.jks
*.keystore

# OS
.DS_Store
`
