import random
import pymysql
from faker import Faker
from config import DB_CONFIG, USER_COUNT

# 初始化Faker
fake = Faker('zh_CN')

# 登录状态
login_statuses = [0, 1]  # 0-失败，1-成功

# 省份城市数据（与用户数据保持一致）
provinces_cities = {
    '北京市': ['东城区', '西城区', '朝阳区', '丰台区', '石景山区'],
    '上海市': ['黄浦区', '徐汇区', '长宁区', '静安区', '普陀区'],
    '广东省': ['广州市', '深圳市', '珠海市', '汕头市', '佛山市'],
    '江苏省': ['南京市', '无锡市', '徐州市', '常州市', '苏州市'],
    '浙江省': ['杭州市', '宁波市', '温州市', '嘉兴市', '湖州市'],
    '山东省': ['济南市', '青岛市', '淄博市', '枣庄市', '东营市'],
    '河南省': ['郑州市', '开封市', '洛阳市', '平顶山市', '安阳市'],
    '河北省': ['石家庄市', '唐山市', '秦皇岛市', '邯郸市', '邢台市'],
    '四川省': ['成都市', '自贡市', '攀枝花市', '泸州市', '德阳市'],
    '湖南省': ['长沙市', '株洲市', '湘潭市', '衡阳市', '邵阳市']
}

def generate_login_history():
    """生成登录历史数据"""
    # 连接数据库
    connection = pymysql.connect(**DB_CONFIG)
    try:
        with connection.cursor() as cursor:
            # 获取用户信息
            cursor.execute("SELECT id, username FROM user_auth LIMIT 50000")
            users = cursor.fetchall()
            
            if not users:
                print("没有找到用户数据，请先生成用户数据")
                return
            
            total_records = 0
            
            # 为每个用户生成登录历史记录
            for user_id, username in users:
                # 每个用户生成1-50条登录记录
                login_count = random.randint(1, 50)
                
                for _ in range(login_count):
                    login_time = fake.date_time_between(start_date='-2y', end_date='now')
                    login_ip = fake.ipv4()
                    user_agent = fake.user_agent()
                    login_status = random.choices(login_statuses, weights=[10, 90], k=1)[0]  # 10%失败，90%成功
                    
                    # 90%的成功登录有地区信息
                    if login_status == 1 and random.random() > 0.1:
                        province = random.choice(list(provinces_cities.keys()))
                        city = random.choice(provinces_cities[province])
                    else:
                        province = None
                        city = None
                    
                    created_at = login_time
                    
                    # 插入登录历史数据
                    login_sql = """
                    INSERT INTO user_login_history (user_id, username, login_time, login_ip, user_agent, 
                                                  login_status, province, city, created_at)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
                    """
                    cursor.execute(login_sql, (
                        user_id, username, login_time, login_ip, user_agent,
                        login_status, province, city, created_at
                    ))
                    
                    total_records += 1
                
                # 每1000个用户提交一次
                if (users.index((user_id, username)) + 1) % 1000 == 0:
                    connection.commit()
                    print(f"已处理 {users.index((user_id, username)) + 1} 个用户的登录历史数据")
            
            # 最后提交
            connection.commit()
            print(f"登录历史数据生成完成，共 {total_records} 条记录")
            
    except Exception as e:
        print(f"生成登录历史数据时出错: {e}")
        connection.rollback()
    finally:
        connection.close()

if __name__ == "__main__":
    generate_login_history()