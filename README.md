Toolchain for building lines schedules for the businfo Android application.

Create a `local.properties` file by setting a path to the bus line schedules:

    lines.dir=/path/to/businfo-sample-lines

Then compile the line schedules with:

    make clean
    make
