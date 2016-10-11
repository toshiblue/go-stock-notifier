CREATE TABLE IF NOT EXISTS stock_data (
id integer PRIMARY KEY,
symbol text, 
exchange text, 
name text, 
change real, 
close real, 
percentagechange real, 
closechange real,
open real, 
high real, 
low real, 
volume real, 
avgvolume real, 
high52 real, 
low52 real, 
marketcap real, 
eps real, 
shares real, 
beta real,
time integer, 
minute integer, 
hour integer, 
day integer, 
month integer, 
year integer
);
