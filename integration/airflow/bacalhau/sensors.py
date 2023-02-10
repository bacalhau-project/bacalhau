# from airflow.sensors.base_sensor_operator import BaseSensorOperator


# class SnowflakeSqlSensor(BaseSensorOperator):
#  def __init__(self, sql *kwargs):
#   self.sql = sql
#   super().__init__(*kwargs)
# def get_connection():
#   ## Create connection to your snowflake database
#   conn = 'conn'
#   return conn
# def poke(self, context):
#   self.snowflake_conn = self.get_connection()
#   response = self.snowflake_conn.execute(self.sql).fetchall()
#   if not response:
#      return False
#   else:
#      if str(response[0][0]) in ('0', '',):
#         return False
#      else:
#         return True