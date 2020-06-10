package models

type TDaoMessage struct {
	Id              int64  `xorm:"pk comment('Primary Key') BIGINT(64)"`
	Height          int64  `xorm:"not null comment('Block Height') index BIGINT(20)"`
	Category        int    `xorm:"not null comment('Message Category: 1.be member 2.add new member 3.create organization 4.been removed 5.remove member 6.be president 7.change president 8.transfer asset 9.issue asset 10.modify organization logo 11.modify organization name 12.new vote 13.proposal rejected 14.proposal expired 15.invited 16.close organization 17.receive asset 18.mint asset') INT(4)"`
	Type            int    `xorm:"not null comment('Message Type: 1.readonly 2.direct execute 3.vote') INT(4)"`
	MessagePosition int    `xorm:"not null comment('Message Scope: 0.official website 1.dao organization 2.all') INT(4)"`
	ContractAddress string `xorm:"not null comment('Contract Address') VARCHAR(64)"`
	Receiver        string `xorm:"comment('Receiver Address') VARCHAR(64)"`
	AdditionalInfo  string `xorm:"not null comment('Additional Information(JSON Format)') VARCHAR(1024)"`
	State           int    `xorm:"not null comment('Message Status: 1.unread 2.read 3.disagree 4.agree') INT(4)"`
	CreateTime      int64  `xorm:"not null comment('Create Time') BIGINT(20)"`
	UpdateTime      int64  `xorm:"not null comment('Update Time') BIGINT(20)"`
}
