-- 简单过程示例
CREATE OR REPLACE PROCEDURE insert_employee(
  p_emp_id IN NUMBER,
  p_emp_name IN VARCHAR2,
  p_department IN VARCHAR2,
  p_salary IN NUMBER
) IS
  v_count NUMBER;
BEGIN
  -- 检查员工是否已存在
  SELECT COUNT(*) INTO v_count
  FROM employees
  WHERE emp_id = p_emp_id;
  
  IF v_count > 0 THEN
    RAISE_APPLICATION_ERROR(-20001, 'Employee already exists');
  END IF;
  
  -- 插入新员工
  INSERT INTO employees (emp_id, emp_name, department, salary, hire_date)
  VALUES (p_emp_id, p_emp_name, p_department, p_salary, SYSDATE);
  
  COMMIT;
  
  DBMS_OUTPUT.PUT_LINE('Employee inserted successfully: ' || p_emp_name);
  
EXCEPTION
  WHEN OTHERS THEN
    ROLLBACK;
    RAISE;
END insert_employee;
/
