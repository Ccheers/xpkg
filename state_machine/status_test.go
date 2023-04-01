package state_machine

import "testing"

//
//
//
//        ***************************     ***************************         *********      ************************
//      *****************************    ******************************      *********      *************************
//     *****************************     *******************************     *********     *************************
//    *********                         *********                *******    *********     *********
//    ********                          *********               ********    *********     ********
//   ********     ******************   *********  *********************    *********     *********
//   ********     *****************    *********  ********************     *********     ********
//  ********      ****************    *********     ****************      *********     *********
//  ********                          *********      ********             *********     ********
// *********                         *********         ******            *********     *********
// ******************************    *********          *******          *********     *************************
//  ****************************    *********            *******        *********      *************************
//    **************************    *********              ******       *********         *********************
//
//

func TestStateMachine_ChangeState(t *testing.T) {
	type args struct {
		from uint
		to   uint
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "1",
			args: args{
				from: 1,
				to:   7,
			},
			wantErr: false,
		},
		{
			name: "2",
			args: args{
				from: 1,
				to:   5,
			},
			wantErr: true,
		},
		{
			name: "3",
			args: args{
				from: 2,
				to:   5,
			},
			wantErr: true,
		},
	}
	x := NewStateMachine()
	x.Register(NewStateNode(1, ""), NewStateNode(3, ""))
	x.Register(NewStateNode(1, ""), NewStateNode(7, ""))
	x.Register(NewStateNode(4, ""), NewStateNode(5, ""))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := x.ChangeState(tt.args.from, tt.args.to); (err != nil) != tt.wantErr {
				t.Errorf("ChangeState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
