-- +goose Up

CREATE TABLE `table_a` (
    `column_1` DATETIME DEFAULT NOW(),
    `column_2` DATETIME DEFAULT NOW(),
    `column_3` DATETIME DEFAULT NOW()
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE `table_b` (
    `column_1` DATETIME DEFAULT NOW(),
    `column_2` DATETIME DEFAULT NOW(),
    `column_3` DATETIME DEFAULT NOW()
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE `table_c` (
    `column_1` DATETIME DEFAULT NOW(),
    `column_2` DATETIME DEFAULT NOW(),
    `column_3` DATETIME DEFAULT NOW()
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

/*!80031 ALTER TABLE `table_a` MODIFY `column_1` TEXT NOT NULL */;
/*!80031 ALTER TABLE `table_b` MODIFY `column_2` TEXT NOT NULL */;
/*!80033 ALTER TABLE `table_c` MODIFY `column_3` TEXT NOT NULL */;
