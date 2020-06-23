<!--
    This Source Code Form is subject to the terms of the Mozilla Public
    License, v. 2.0. If a copy of the MPL was not distributed with this
    file, You can obtain one at http://mozilla.org/MPL/2.0/.
-->

<!--
    Copyright 2020 Joyent, Inc.
-->

# eng: Joyent Engineering Guide

This repository is part of the Joyent Triton project. See the [contribution
guidelines][2] and general documentation at the main
[Triton project][1] page.


After the boilerplate paragraph, write a brief description about your repo.


## Development

To ensure maximum compatibility, release builds are performed on a build zone
that is old enough to allow new and updated components to run on all supported
platform images.  If you are not using the Joyent Jenkins instance for
performing builds, you should build using an appropriate build zone.  See
[Build Zone Setup for Manta and Triton](https://github.com/joyent/triton/blob/master/docs/developer-guide/build-zone-setup.md).

Describe steps necessary for development here.

    make all


## Test

Describe steps necessary for testing here.

    make test


## Documentation

[See docs/readme.md](docs/readme.md).

To update, edit "docs/readme.md" and run `make docs`
to update "docs/readme.html". Works on either SmartOS or Mac OS X.


## License

"triton-shim: Joyent Triton" is licensed under the
[Mozilla Public License version 2.0](http://mozilla.org/MPL/2.0/).
See the file LICENSE.

[1]: https://github.com/joyent/triton
[2]: https://github.com/joyent/triton/blob/master/CONTRIBUTING.md
