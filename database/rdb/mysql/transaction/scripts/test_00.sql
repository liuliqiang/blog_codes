-- 创建shop库 
-- CREATE DATABASE lq_test CHARACTER SET utf8 COLLATE utf8_general_ci 

-- 使用测试库 
USE lq_test;

-- 创建表 
CREATE TABLE IF NOT EXISTS `lq_test`.`account`( 
    `id` INT(3) NOT NULL AUTO_INCREMENT, 
    `name` VARCHAR(30) NOT NULL, 
    `money` DECIMAL(9,2) NOT NULL, PRIMARY KEY (`id`) 
)ENGINE = INNODB DEFAULT CHARSET=utf8 ;

INSERT INTO `account` (`id`, `name`,`money`) 
VALUES (1, 'A',2000.00)
ON DUPLICATE KEY UPDATE
    money = 2000.00;

INSERT INTO `account` (`id`, `name`,`money`) 
VALUES (2, 'B',10000.00)
ON DUPLICATE KEY UPDATE
    money = 10000.00;

        
-- =================== 模拟转账：事务 =========================== 

SET autocommit = 0; 
-- 关闭自动提交 

START TRANSACTION;
-- 开启一个事务（一组事务） 
-- 执行 
UPDATE `account` SET `money`=`money`-500 WHERE `name` = 'A'; /* A减五百 */
UPDATE `account` SET money=money+500 WHERE `name` = 'B'; -- B加五百 
-- 执行完毕 
COMMIT; -- 提交 
ROLLBACK; -- 回滚 
SET autocommit = 1; -- 打开自动提交