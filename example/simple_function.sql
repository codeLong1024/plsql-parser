-- 简单函数示例
CREATE OR REPLACE FUNCTION calculate_tax(
  p_amount IN NUMBER,
  p_tax_rate IN NUMBER DEFAULT 0.13
) RETURN NUMBER IS
  v_tax_amount NUMBER;
BEGIN
  IF p_amount <= 0 THEN
    RETURN 0;
  END IF;
  
  v_tax_amount := p_amount * p_tax_rate;
  
  DBMS_OUTPUT.PUT_LINE('Tax calculated: ' || v_tax_amount);
  
  RETURN ROUND(v_tax_amount, 2);
END calculate_tax;
/
