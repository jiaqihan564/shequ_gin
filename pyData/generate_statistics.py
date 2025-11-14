import random
import pymysql
from faker import Faker
from config import DB_CONFIG
from datetime import datetime, timedelta

# 初始化Faker
fake = Faker()

def generate_statistics():
    """生成统计数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 生成每日指标数据（过去2年的数据）
            print("正在生成每日指标数据...")
            start_date = datetime.now() - timedelta(days=730)  # 2年前
            end_date = datetime.now()
            
            current_date = start_date
            daily_metrics_count = 0
            
            while current_date <= end_date:
                active_users = random.randint(100, 10000)
                avg_response_time = round(random.uniform(50, 500), 2)
                success_rate = round(random.uniform(90, 99.99), 2)
                peak_concurrent = random.randint(10, 1000)
                most_popular_endpoint = random.choice([
                    '/api/users/login', '/api/articles/list', '/api/resources/list',
                    '/api/chat/messages', '/api/code/execute'
                ])
                new_users = random.randint(10, 500)
                total_requests = random.randint(1000, 50000)
                created_at = current_date
                updated_at = current_date
                
                # 插入每日指标数据
                daily_metrics_sql = """
                INSERT INTO daily_metrics (date, active_users, avg_response_time, success_rate,
                                        peak_concurrent, most_popular_endpoint, new_users,
                                        total_requests, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                """
                cursor.execute(daily_metrics_sql, (
                    current_date.date(), active_users, avg_response_time, success_rate,
                    peak_concurrent, most_popular_endpoint, new_users,
                    total_requests, created_at, updated_at
                ))
                
                current_date += timedelta(days=1)
                daily_metrics_count += 1
                
                # 每100条提交一次
                if daily_metrics_count % 100 == 0:
                    connection.commit()
                    print(f"已插入 {daily_metrics_count} 条每日指标数据")
            
            connection.commit()
            print(f"每日指标数据生成完成，共 {daily_metrics_count} 条记录")
            
            # 生成API统计数据
            print("正在生成API统计数据...")
            api_endpoints = [
                '/api/users/login', '/api/users/register', '/api/articles/list', '/api/articles/detail',
                '/api/resources/list', '/api/resources/detail', '/api/chat/messages', '/api/code/execute',
                '/api/users/profile', '/api/articles/create', '/api/resources/upload'
            ]
            methods = ['GET', 'POST', 'PUT', 'DELETE']
            
            api_stats_count = 0
            
            # 为过去30天生成API统计数据
            for i in range(30):
                date = (datetime.now() - timedelta(days=i)).date()
                
                # 为每个端点生成数据
                for endpoint in api_endpoints:
                    for method in methods:
                        success_count = random.randint(100, 10000)
                        error_count = random.randint(0, 1000)
                        total_count = success_count + error_count
                        avg_latency_ms = round(random.uniform(50, 1000), 2)
                        created_at = datetime.now()
                        updated_at = datetime.now()
                        
                        # 插入API统计数据
                        api_stats_sql = """
                        INSERT INTO api_statistics (date, endpoint, method, success_count, error_count,
                                                 total_count, avg_latency_ms, created_at, updated_at)
                        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                        """
                        cursor.execute(api_stats_sql, (
                            date, endpoint, method, success_count, error_count,
                            total_count, avg_latency_ms, created_at, updated_at
                        ))
                        
                        api_stats_count += 1
                
                # 每5天提交一次
                if (i + 1) % 5 == 0:
                    connection.commit()
                    print(f"已处理 {i + 1} 天的API统计数据")
            
            connection.commit()
            print(f"API统计数据生成完成，共 {api_stats_count} 条记录")
            
            # 生成用户统计数据
            print("正在生成用户统计数据...")
            user_stats_count = 0
            
            # 为过去365天生成用户统计数据
            for i in range(365):
                date = (datetime.now() - timedelta(days=i)).date()
                login_count = random.randint(100, 5000)
                register_count = random.randint(10, 500)
                created_at = datetime.now()
                updated_at = datetime.now()
                
                # 插入用户统计数据
                user_stats_sql = """
                INSERT INTO user_statistics (date, login_count, register_count, created_at, updated_at)
                VALUES (%s, %s, %s, %s, %s)
                """
                cursor.execute(user_stats_sql, (
                    date, login_count, register_count, created_at, updated_at
                ))
                
                user_stats_count += 1
                
                # 每50天提交一次
                if (i + 1) % 50 == 0:
                    connection.commit()
                    print(f"已处理 {i + 1} 天的用户统计数据")
            
            connection.commit()
            print(f"用户统计数据生成完成，共 {user_stats_count} 条记录")
            
            # 更新累计统计数据
            print("正在更新累计统计数据...")
            
            # 更新总用户数
            cursor.execute("SELECT COUNT(*) FROM user_auth")
            total_users = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_users'
            """, (total_users,))
            
            # 更新总文章数
            cursor.execute("SELECT COUNT(*) FROM articles WHERE status = 1")
            total_articles = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_articles'
            """, (total_articles,))
            
            # 更新总资源数
            cursor.execute("SELECT COUNT(*) FROM resources WHERE status = 1")
            total_resources = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_resources'
            """, (total_resources,))
            
            # 更新总代码片段数
            cursor.execute("SELECT COUNT(*) FROM code_snippets")
            total_code_snippets = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_code_snippets'
            """, (total_code_snippets,))
            
            # 更新总聊天消息数
            cursor.execute("SELECT COUNT(*) FROM chat_messages WHERE status = 1")
            total_chat_messages = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_chat_messages'
            """, (total_chat_messages,))
            
            # 更新总API调用次数
            cursor.execute("SELECT SUM(total_count) FROM api_statistics")
            total_api_calls = cursor.fetchone()[0] or 0
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_api_calls'
            """, (total_api_calls,))
            
            # 更新总评论数
            cursor.execute("""
                SELECT 
                    (SELECT COUNT(*) FROM article_comments WHERE status = 1) +
                    (SELECT COUNT(*) FROM resource_comments WHERE status = 1)
            """)
            total_comments = cursor.fetchone()[0] or 0
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_comments'
            """, (total_comments,))
            
            # 更新总登录次数
            cursor.execute("SELECT COUNT(*) FROM user_login_history WHERE login_status = 1")
            total_logins = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_logins'
            """, (total_logins,))
            
            # 更新总注册用户数
            cursor.execute("SELECT COUNT(*) FROM user_auth")
            total_registrations = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'total_registrations'
            """, (total_registrations,))
            
            # 更新今日活跃用户数（近24小时登录的用户数）
            cursor.execute("""
                SELECT COUNT(DISTINCT user_id) 
                FROM user_login_history 
                WHERE login_status = 1 AND login_time >= NOW() - INTERVAL 1 DAY
            """)
            active_users_today = cursor.fetchone()[0]
            cursor.execute("""
                UPDATE cumulative_statistics 
                SET stat_value = %s 
                WHERE stat_key = 'active_users_today'
            """, (active_users_today,))
            
            # 最后提交
            connection.commit()
            print("累计统计数据更新完成")
            print("统计数据生成完成")
            
    except Exception as e:
        print(f"生成统计数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_statistics()