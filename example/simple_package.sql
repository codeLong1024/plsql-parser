CREATE OR REPLACE PACKAGE example_pkg IS
  -- 包规格示例
  g_version VARCHAR2(20) := '1.0';
  
  FUNCTION get_user_name(p_user_id IN NUMBER) RETURN VARCHAR2;
  
  PROCEDURE process_order(p_order_id IN NUMBER, p_status OUT VARCHAR2);
END example_pkg;
/

CREATE OR REPLACE PACKAGE BODY example_pkg IS
  
  FUNCTION get_user_name(p_user_id IN NUMBER) RETURN VARCHAR2 IS
    v_user_name VARCHAR2(100);
  BEGIN
    SELECT user_name INTO v_user_name
    FROM fnd_user
    WHERE user_id = p_user_id;
    
    RETURN v_user_name;
  EXCEPTION
    WHEN NO_DATA_FOUND THEN
      RETURN NULL;
  END get_user_name;
  
  PROCEDURE process_order(p_order_id IN NUMBER, p_status OUT VARCHAR2) IS
    v_order_count NUMBER;
  BEGIN
    SELECT COUNT(*) INTO v_order_count
    FROM orders
    WHERE order_id = p_order_id;
    
    IF v_order_count > 0 THEN
      UPDATE orders
      SET status = 'PROCESSED'
      WHERE order_id = p_order_id;
      
      p_status := 'SUCCESS';
    ELSE
      p_status := 'NOT_FOUND';
    END IF;
    
    COMMIT;
  EXCEPTION
    WHEN OTHERS THEN
      ROLLBACK;
      p_status := 'ERROR: ' || SQLERRM;
  END process_order;
  
END example_pkg;
/
