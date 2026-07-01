SELECT
    customer_id,
    customer_name,
    SUM(order_amount) AS total_spent
FROM orders
WHERE created_at >= NOW() - INTERVAL '90 days'
GROUP BY customer_id, customer_name
ORDER BY total_spent DESC
LIMIT 20;
