package av1

func invDct41dInternal(c []int32, cOff, stride int, min, max int32, tx64 bool) {
	clip_ := func(v int32) int32 { return clip(v, min, max) }
	in0, in1 := c[cOff+0*stride], c[cOff+1*stride]
	var t0, t1, t2, t3 int32
	if tx64 {
		t1 = (in0*181 + 128) >> 8
		t0 = t1
		t2 = (in1*1567 + 2048) >> 12
		t3 = (in1*3784 + 2048) >> 12
	} else {
		in2, in3 := c[cOff+2*stride], c[cOff+3*stride]
		t0 = ((in0+in2)*181 + 128) >> 8
		t1 = ((in0-in2)*181 + 128) >> 8
		t2 = ((in1*1567 - in3*(3784-4096) + 2048) >> 12) - in3
		t3 = ((in1*(3784-4096) + in3*1567 + 2048) >> 12) + in1
	}
	c[cOff+0*stride] = clip_(t0 + t3)
	c[cOff+1*stride] = clip_(t1 + t2)
	c[cOff+2*stride] = clip_(t1 - t2)
	c[cOff+3*stride] = clip_(t0 - t3)
}

func invDct41d(c []int32, cOff, stride int, min, max int32) {
	invDct41dInternal(c, cOff, stride, min, max, false)
}

func invDct81dInternal(c []int32, cOff, stride int, min, max int32, tx64 bool) {
	clip_ := func(v int32) int32 { return clip(v, min, max) }
	invDct41dInternal(c, cOff, stride<<1, min, max, tx64)
	in1, in3 := c[cOff+1*stride], c[cOff+3*stride]
	var t4a, t5a, t6a, t7a int32
	if tx64 {
		t4a = (in1*799 + 2048) >> 12
		t5a = (in3*-2276 + 2048) >> 12
		t6a = (in3*3406 + 2048) >> 12
		t7a = (in1*4017 + 2048) >> 12
	} else {
		in5, in7 := c[cOff+5*stride], c[cOff+7*stride]
		t4a = ((in1*799 - in7*(4017-4096) + 2048) >> 12) - in7
		t5a = (in5*1703 - in3*1138 + 1024) >> 11
		t6a = (in5*1138 + in3*1703 + 1024) >> 11
		t7a = ((in1*(4017-4096) + in7*799 + 2048) >> 12) + in1
	}
	t4 := clip_(t4a + t5a)
	t5a = clip_(t4a - t5a)
	t7 := clip_(t7a + t6a)
	t6a = clip_(t7a - t6a)
	t5 := ((t6a-t5a)*181 + 128) >> 8
	t6 := ((t6a+t5a)*181 + 128) >> 8
	t0 := c[cOff+0*stride]
	t1 := c[cOff+2*stride]
	t2 := c[cOff+4*stride]
	t3 := c[cOff+6*stride]
	c[cOff+0*stride] = clip_(t0 + t7)
	c[cOff+1*stride] = clip_(t1 + t6)
	c[cOff+2*stride] = clip_(t2 + t5)
	c[cOff+3*stride] = clip_(t3 + t4)
	c[cOff+4*stride] = clip_(t3 - t4)
	c[cOff+5*stride] = clip_(t2 - t5)
	c[cOff+6*stride] = clip_(t1 - t6)
	c[cOff+7*stride] = clip_(t0 - t7)
}

func invDct81d(c []int32, cOff, stride int, min, max int32) {
	invDct81dInternal(c, cOff, stride, min, max, false)
}

func invDct161dInternal(c []int32, cOff, stride int, min, max int32, tx64 bool) {
	clip_ := func(v int32) int32 { return clip(v, min, max) }
	invDct81dInternal(c, cOff, stride<<1, min, max, tx64)
	in1, in3 := c[cOff+1*stride], c[cOff+3*stride]
	in5, in7 := c[cOff+5*stride], c[cOff+7*stride]
	var t8a, t9a, t10a, t11a, t12a, t13a, t14a, t15a int32
	if tx64 {
		t8a = (in1*401 + 2048) >> 12
		t9a = (in7*-2598 + 2048) >> 12
		t10a = (in5*1931 + 2048) >> 12
		t11a = (in3*-1189 + 2048) >> 12
		t12a = (in3*3920 + 2048) >> 12
		t13a = (in5*3612 + 2048) >> 12
		t14a = (in7*3166 + 2048) >> 12
		t15a = (in1*4076 + 2048) >> 12
	} else {
		in9, in11 := c[cOff+9*stride], c[cOff+11*stride]
		in13, in15 := c[cOff+13*stride], c[cOff+15*stride]
		t8a = ((in1*401 - in15*(4076-4096) + 2048) >> 12) - in15
		t9a = (in9*1583 - in7*1299 + 1024) >> 11
		t10a = ((in5*1931 - in11*(3612-4096) + 2048) >> 12) - in11
		t11a = ((in13*(3920-4096) - in3*1189 + 2048) >> 12) + in13
		t12a = ((in13*1189 + in3*(3920-4096) + 2048) >> 12) + in3
		t13a = ((in5*(3612-4096) + in11*1931 + 2048) >> 12) + in5
		t14a = (in9*1299 + in7*1583 + 1024) >> 11
		t15a = ((in1*(4076-4096) + in15*401 + 2048) >> 12) + in1
	}
	t8 := clip_(t8a + t9a)
	t9 := clip_(t8a - t9a)
	t10 := clip_(t11a - t10a)
	t11 := clip_(t11a + t10a)
	t12 := clip_(t12a + t13a)
	t13 := clip_(t12a - t13a)
	t14 := clip_(t15a - t14a)
	t15 := clip_(t15a + t14a)
	t9a = ((t14*1567 - t9*(3784-4096) + 2048) >> 12) - t9
	t14a = ((t14*(3784-4096) + t9*1567 + 2048) >> 12) + t14
	t10a = ((-(t13*(3784-4096) + t10*1567) + 2048) >> 12) - t13
	t13a = ((t13*1567 - t10*(3784-4096) + 2048) >> 12) - t10
	t8a = clip_(t8 + t11)
	t9 = clip_(t9a + t10a)
	t10 = clip_(t9a - t10a)
	t11a = clip_(t8 - t11)
	t12a = clip_(t15 - t12)
	t13 = clip_(t14a - t13a)
	t14 = clip_(t14a + t13a)
	t15a = clip_(t15 + t12)
	t10a = ((t13-t10)*181 + 128) >> 8
	t13a = ((t13+t10)*181 + 128) >> 8
	t11 = ((t12a-t11a)*181 + 128) >> 8
	t12 = ((t12a+t11a)*181 + 128) >> 8
	t0 := c[cOff+0*stride]
	t1 := c[cOff+2*stride]
	t2 := c[cOff+4*stride]
	t3 := c[cOff+6*stride]
	t4 := c[cOff+8*stride]
	t5 := c[cOff+10*stride]
	t6 := c[cOff+12*stride]
	t7 := c[cOff+14*stride]
	c[cOff+0*stride] = clip_(t0 + t15a)
	c[cOff+1*stride] = clip_(t1 + t14)
	c[cOff+2*stride] = clip_(t2 + t13a)
	c[cOff+3*stride] = clip_(t3 + t12)
	c[cOff+4*stride] = clip_(t4 + t11)
	c[cOff+5*stride] = clip_(t5 + t10a)
	c[cOff+6*stride] = clip_(t6 + t9)
	c[cOff+7*stride] = clip_(t7 + t8a)
	c[cOff+8*stride] = clip_(t7 - t8a)
	c[cOff+9*stride] = clip_(t6 - t9)
	c[cOff+10*stride] = clip_(t5 - t10a)
	c[cOff+11*stride] = clip_(t4 - t11)
	c[cOff+12*stride] = clip_(t3 - t12)
	c[cOff+13*stride] = clip_(t2 - t13a)
	c[cOff+14*stride] = clip_(t1 - t14)
	c[cOff+15*stride] = clip_(t0 - t15a)
}

func invDct161d(c []int32, cOff, stride int, min, max int32) {
	invDct161dInternal(c, cOff, stride, min, max, false)
}

func invDct321dInternal(c []int32, cOff, stride int, min, max int32, tx64 bool) {
	clip_ := func(v int32) int32 { return clip(v, min, max) }
	invDct161dInternal(c, cOff, stride<<1, min, max, tx64)
	in1, in3 := c[cOff+1*stride], c[cOff+3*stride]
	in5, in7 := c[cOff+5*stride], c[cOff+7*stride]
	in9, in11 := c[cOff+9*stride], c[cOff+11*stride]
	in13, in15 := c[cOff+13*stride], c[cOff+15*stride]
	var t16a, t17a, t18a, t19a, t20a, t21a, t22a, t23a int32
	var t24a, t25a, t26a, t27a, t28a, t29a, t30a, t31a int32
	if tx64 {
		t16a = (in1*201 + 2048) >> 12
		t17a = (in15*-2751 + 2048) >> 12
		t18a = (in9*1751 + 2048) >> 12
		t19a = (in7*-1380 + 2048) >> 12
		t20a = (in5*995 + 2048) >> 12
		t21a = (in11*-2106 + 2048) >> 12
		t22a = (in13*2440 + 2048) >> 12
		t23a = (in3*-601 + 2048) >> 12
		t24a = (in3*4052 + 2048) >> 12
		t25a = (in13*3290 + 2048) >> 12
		t26a = (in11*3513 + 2048) >> 12
		t27a = (in5*3973 + 2048) >> 12
		t28a = (in7*3857 + 2048) >> 12
		t29a = (in9*3703 + 2048) >> 12
		t30a = (in15*3035 + 2048) >> 12
		t31a = (in1*4091 + 2048) >> 12
	} else {
		in17, in19 := c[cOff+17*stride], c[cOff+19*stride]
		in21, in23 := c[cOff+21*stride], c[cOff+23*stride]
		in25, in27 := c[cOff+25*stride], c[cOff+27*stride]
		in29, in31 := c[cOff+29*stride], c[cOff+31*stride]
		t16a = ((in1*201 - in31*(4091-4096) + 2048) >> 12) - in31
		t17a = ((in17*(3035-4096) - in15*2751 + 2048) >> 12) + in17
		t18a = ((in9*1751 - in23*(3703-4096) + 2048) >> 12) - in23
		t19a = ((in25*(3857-4096) - in7*1380 + 2048) >> 12) + in25
		t20a = ((in5*995 - in27*(3973-4096) + 2048) >> 12) - in27
		t21a = ((in21*(3513-4096) - in11*2106 + 2048) >> 12) + in21
		t22a = (in13*1220 - in19*1645 + 1024) >> 11
		t23a = ((in29*(4052-4096) - in3*601 + 2048) >> 12) + in29
		t24a = ((in29*601 + in3*(4052-4096) + 2048) >> 12) + in3
		t25a = (in13*1645 + in19*1220 + 1024) >> 11
		t26a = ((in21*2106 + in11*(3513-4096) + 2048) >> 12) + in11
		t27a = ((in5*(3973-4096) + in27*995 + 2048) >> 12) + in5
		t28a = ((in25*1380 + in7*(3857-4096) + 2048) >> 12) + in7
		t29a = ((in9*(3703-4096) + in23*1751 + 2048) >> 12) + in9
		t30a = ((in17*2751 + in15*(3035-4096) + 2048) >> 12) + in15
		t31a = ((in1*(4091-4096) + in31*201 + 2048) >> 12) + in1
	}
	t16 := clip_(t16a + t17a)
	t17 := clip_(t16a - t17a)
	t18 := clip_(t19a - t18a)
	t19 := clip_(t19a + t18a)
	t20 := clip_(t20a + t21a)
	t21 := clip_(t20a - t21a)
	t22 := clip_(t23a - t22a)
	t23 := clip_(t23a + t22a)
	t24 := clip_(t24a + t25a)
	t25 := clip_(t24a - t25a)
	t26 := clip_(t27a - t26a)
	t27 := clip_(t27a + t26a)
	t28 := clip_(t28a + t29a)
	t29 := clip_(t28a - t29a)
	t30 := clip_(t31a - t30a)
	t31 := clip_(t31a + t30a)
	t17a = ((t30*799 - t17*(4017-4096) + 2048) >> 12) - t17
	t30a = ((t30*(4017-4096) + t17*799 + 2048) >> 12) + t30
	t18a = ((-(t29*(4017-4096) + t18*799) + 2048) >> 12) - t29
	t29a = ((t29*799 - t18*(4017-4096) + 2048) >> 12) - t18
	t21a = (t26*1703 - t21*1138 + 1024) >> 11
	t26a = (t26*1138 + t21*1703 + 1024) >> 11
	t22a = (-(t25*1138 + t22*1703) + 1024) >> 11
	t25a = (t25*1703 - t22*1138 + 1024) >> 11
	t16a = clip_(t16 + t19)
	t17 = clip_(t17a + t18a)
	t18 = clip_(t17a - t18a)
	t19a = clip_(t16 - t19)
	t20a = clip_(t23 - t20)
	t21 = clip_(t22a - t21a)
	t22 = clip_(t22a + t21a)
	t23a = clip_(t23 + t20)
	t24a = clip_(t24 + t27)
	t25 = clip_(t25a + t26a)
	t26 = clip_(t25a - t26a)
	t27a = clip_(t24 - t27)
	t28a = clip_(t31 - t28)
	t29 = clip_(t30a - t29a)
	t30 = clip_(t30a + t29a)
	t31a = clip_(t31 + t28)
	t18a = ((t29*1567 - t18*(3784-4096) + 2048) >> 12) - t18
	t29a = ((t29*(3784-4096) + t18*1567 + 2048) >> 12) + t29
	t19 = ((t28a*1567 - t19a*(3784-4096) + 2048) >> 12) - t19a
	t28 = ((t28a*(3784-4096) + t19a*1567 + 2048) >> 12) + t28a
	t20 = ((-(t27a*(3784-4096) + t20a*1567) + 2048) >> 12) - t27a
	t27 = ((t27a*1567 - t20a*(3784-4096) + 2048) >> 12) - t20a
	t21a = ((-(t26*(3784-4096) + t21*1567) + 2048) >> 12) - t26
	t26a = ((t26*1567 - t21*(3784-4096) + 2048) >> 12) - t21
	t16 = clip_(t16a + t23a)
	t17a = clip_(t17 + t22)
	t18 = clip_(t18a + t21a)
	t19a = clip_(t19 + t20)
	t20a = clip_(t19 - t20)
	t21 = clip_(t18a - t21a)
	t22a = clip_(t17 - t22)
	t23 = clip_(t16a - t23a)
	t24 = clip_(t31a - t24a)
	t25a = clip_(t30 - t25)
	t26 = clip_(t29a - t26a)
	t27a = clip_(t28 - t27)
	t28a = clip_(t28 + t27)
	t29 = clip_(t29a + t26a)
	t30a = clip_(t30 + t25)
	t31 = clip_(t31a + t24a)
	t20 = ((t27a-t20a)*181 + 128) >> 8
	t27 = ((t27a+t20a)*181 + 128) >> 8
	t21a = ((t26-t21)*181 + 128) >> 8
	t26a = ((t26+t21)*181 + 128) >> 8
	t22 = ((t25a-t22a)*181 + 128) >> 8
	t25 = ((t25a+t22a)*181 + 128) >> 8
	t23a = ((t24-t23)*181 + 128) >> 8
	t24a = ((t24+t23)*181 + 128) >> 8
	t0 := c[cOff+0*stride]
	t1 := c[cOff+2*stride]
	t2 := c[cOff+4*stride]
	t3 := c[cOff+6*stride]
	t4 := c[cOff+8*stride]
	t5 := c[cOff+10*stride]
	t6 := c[cOff+12*stride]
	t7 := c[cOff+14*stride]
	t8 := c[cOff+16*stride]
	t9 := c[cOff+18*stride]
	t10 := c[cOff+20*stride]
	t11 := c[cOff+22*stride]
	t12 := c[cOff+24*stride]
	t13 := c[cOff+26*stride]
	t14 := c[cOff+28*stride]
	t15 := c[cOff+30*stride]
	c[cOff+0*stride] = clip_(t0 + t31)
	c[cOff+1*stride] = clip_(t1 + t30a)
	c[cOff+2*stride] = clip_(t2 + t29)
	c[cOff+3*stride] = clip_(t3 + t28a)
	c[cOff+4*stride] = clip_(t4 + t27)
	c[cOff+5*stride] = clip_(t5 + t26a)
	c[cOff+6*stride] = clip_(t6 + t25)
	c[cOff+7*stride] = clip_(t7 + t24a)
	c[cOff+8*stride] = clip_(t8 + t23a)
	c[cOff+9*stride] = clip_(t9 + t22)
	c[cOff+10*stride] = clip_(t10 + t21a)
	c[cOff+11*stride] = clip_(t11 + t20)
	c[cOff+12*stride] = clip_(t12 + t19a)
	c[cOff+13*stride] = clip_(t13 + t18)
	c[cOff+14*stride] = clip_(t14 + t17a)
	c[cOff+15*stride] = clip_(t15 + t16)
	c[cOff+16*stride] = clip_(t15 - t16)
	c[cOff+17*stride] = clip_(t14 - t17a)
	c[cOff+18*stride] = clip_(t13 - t18)
	c[cOff+19*stride] = clip_(t12 - t19a)
	c[cOff+20*stride] = clip_(t11 - t20)
	c[cOff+21*stride] = clip_(t10 - t21a)
	c[cOff+22*stride] = clip_(t9 - t22)
	c[cOff+23*stride] = clip_(t8 - t23a)
	c[cOff+24*stride] = clip_(t7 - t24a)
	c[cOff+25*stride] = clip_(t6 - t25)
	c[cOff+26*stride] = clip_(t5 - t26a)
	c[cOff+27*stride] = clip_(t4 - t27)
	c[cOff+28*stride] = clip_(t3 - t28a)
	c[cOff+29*stride] = clip_(t2 - t29)
	c[cOff+30*stride] = clip_(t1 - t30a)
	c[cOff+31*stride] = clip_(t0 - t31)
}

func invDct321d(c []int32, cOff, stride int, min, max int32) {
	invDct321dInternal(c, cOff, stride, min, max, false)
}

func invDct641d(c []int32, cOff, stride int, min, max int32) {
	clip_ := func(v int32) int32 { return clip(v, min, max) }
	invDct321dInternal(c, cOff, stride<<1, min, max, true)
	in1, in3 := c[cOff+1*stride], c[cOff+3*stride]
	in5, in7 := c[cOff+5*stride], c[cOff+7*stride]
	in9, in11 := c[cOff+9*stride], c[cOff+11*stride]
	in13, in15 := c[cOff+13*stride], c[cOff+15*stride]
	in17, in19 := c[cOff+17*stride], c[cOff+19*stride]
	in21, in23 := c[cOff+21*stride], c[cOff+23*stride]
	in25, in27 := c[cOff+25*stride], c[cOff+27*stride]
	in29, in31 := c[cOff+29*stride], c[cOff+31*stride]
	t32a := (in1*101 + 2048) >> 12
	t33a := (in31*-2824 + 2048) >> 12
	t34a := (in17*1660 + 2048) >> 12
	t35a := (in15*-1474 + 2048) >> 12
	t36a := (in9*897 + 2048) >> 12
	t37a := (in23*-2191 + 2048) >> 12
	t38a := (in25*2359 + 2048) >> 12
	t39a := (in7*-700 + 2048) >> 12
	t40a := (in5*501 + 2048) >> 12
	t41a := (in27*-2520 + 2048) >> 12
	t42a := (in21*2019 + 2048) >> 12
	t43a := (in11*-1092 + 2048) >> 12
	t44a := (in13*1285 + 2048) >> 12
	t45a := (in19*-1842 + 2048) >> 12
	t46a := (in29*2675 + 2048) >> 12
	t47a := (in3*-301 + 2048) >> 12
	t48a := (in3*4085 + 2048) >> 12
	t49a := (in29*3102 + 2048) >> 12
	t50a := (in19*3659 + 2048) >> 12
	t51a := (in13*3889 + 2048) >> 12
	t52a := (in11*3948 + 2048) >> 12
	t53a := (in21*3564 + 2048) >> 12
	t54a := (in27*3229 + 2048) >> 12
	t55a := (in5*4065 + 2048) >> 12
	t56a := (in7*4036 + 2048) >> 12
	t57a := (in25*3349 + 2048) >> 12
	t58a := (in23*3461 + 2048) >> 12
	t59a := (in9*3996 + 2048) >> 12
	t60a := (in15*3822 + 2048) >> 12
	t61a := (in17*3745 + 2048) >> 12
	t62a := (in31*2967 + 2048) >> 12
	t63a := (in1*4095 + 2048) >> 12
	t32 := clip_(t32a + t33a)
	t33 := clip_(t32a - t33a)
	t34 := clip_(t35a - t34a)
	t35 := clip_(t35a + t34a)
	t36 := clip_(t36a + t37a)
	t37 := clip_(t36a - t37a)
	t38 := clip_(t39a - t38a)
	t39 := clip_(t39a + t38a)
	t40 := clip_(t40a + t41a)
	t41 := clip_(t40a - t41a)
	t42 := clip_(t43a - t42a)
	t43 := clip_(t43a + t42a)
	t44 := clip_(t44a + t45a)
	t45 := clip_(t44a - t45a)
	t46 := clip_(t47a - t46a)
	t47 := clip_(t47a + t46a)
	t48 := clip_(t48a + t49a)
	t49 := clip_(t48a - t49a)
	t50 := clip_(t51a - t50a)
	t51 := clip_(t51a + t50a)
	t52 := clip_(t52a + t53a)
	t53 := clip_(t52a - t53a)
	t54 := clip_(t55a - t54a)
	t55 := clip_(t55a + t54a)
	t56 := clip_(t56a + t57a)
	t57 := clip_(t56a - t57a)
	t58 := clip_(t59a - t58a)
	t59 := clip_(t59a + t58a)
	t60 := clip_(t60a + t61a)
	t61 := clip_(t60a - t61a)
	t62 := clip_(t63a - t62a)
	t63 := clip_(t63a + t62a)
	t33a = ((t33*(4096-4076) + t62*401 + 2048) >> 12) - t33
	t34a = ((t34*-401 + t61*(4096-4076) + 2048) >> 12) - t61
	t37a = (t37*-1299 + t58*1583 + 1024) >> 11
	t38a = (t38*-1583 + t57*-1299 + 1024) >> 11
	t41a = ((t41*(4096-3612) + t54*1931 + 2048) >> 12) - t41
	t42a = ((t42*-1931 + t53*(4096-3612) + 2048) >> 12) - t53
	t45a = ((t45*-1189 + t50*(3920-4096) + 2048) >> 12) + t50
	t46a = ((t46*(4096-3920) + t49*-1189 + 2048) >> 12) - t46
	t49a = ((t46*-1189 + t49*(3920-4096) + 2048) >> 12) + t49
	t50a = ((t45*(3920-4096) + t50*1189 + 2048) >> 12) + t45
	t53a = ((t42*(4096-3612) + t53*1931 + 2048) >> 12) - t42
	t54a = ((t41*1931 + t54*(3612-4096) + 2048) >> 12) + t54
	t57a = (t38*-1299 + t57*1583 + 1024) >> 11
	t58a = (t37*1583 + t58*1299 + 1024) >> 11
	t61a = ((t34*(4096-4076) + t61*401 + 2048) >> 12) - t34
	t62a = ((t33*401 + t62*(4076-4096) + 2048) >> 12) + t62
	t32a = clip_(t32 + t35)
	t33 = clip_(t33a + t34a)
	t34 = clip_(t33a - t34a)
	t35a = clip_(t32 - t35)
	t36a = clip_(t39 - t36)
	t37 = clip_(t38a - t37a)
	t38 = clip_(t38a + t37a)
	t39a = clip_(t39 + t36)
	t40a = clip_(t40 + t43)
	t41 = clip_(t41a + t42a)
	t42 = clip_(t41a - t42a)
	t43a = clip_(t40 - t43)
	t44a = clip_(t47 - t44)
	t45 = clip_(t46a - t45a)
	t46 = clip_(t46a + t45a)
	t47a = clip_(t47 + t44)
	t48a = clip_(t48 + t51)
	t49 = clip_(t49a + t50a)
	t50 = clip_(t49a - t50a)
	t51a = clip_(t48 - t51)
	t52a = clip_(t55 - t52)
	t53 = clip_(t54a - t53a)
	t54 = clip_(t54a + t53a)
	t55a = clip_(t55 + t52)
	t56a = clip_(t56 + t59)
	t57 = clip_(t57a + t58a)
	t58 = clip_(t57a - t58a)
	t59a = clip_(t56 - t59)
	t60a = clip_(t63 - t60)
	t61 = clip_(t62a - t61a)
	t62 = clip_(t62a + t61a)
	t63a = clip_(t63 + t60)
	t34a = ((t34*(4096-4017) + t61*799 + 2048) >> 12) - t34
	t35 = ((t35a*(4096-4017) + t60a*799 + 2048) >> 12) - t35a
	t36 = ((t36a*-799 + t59a*(4096-4017) + 2048) >> 12) - t59a
	t37a = ((t37*-799 + t58*(4096-4017) + 2048) >> 12) - t58
	t42a = (t42*-1138 + t53*1703 + 1024) >> 11
	t43 = (t43a*-1138 + t52a*1703 + 1024) >> 11
	t44 = (t44a*-1703 + t51a*-1138 + 1024) >> 11
	t45a = (t45*-1703 + t50*-1138 + 1024) >> 11
	t50a = (t45*-1138 + t50*1703 + 1024) >> 11
	t51 = (t44a*-1138 + t51a*1703 + 1024) >> 11
	t52 = (t43a*1703 + t52a*1138 + 1024) >> 11
	t53a = (t42*1703 + t53*1138 + 1024) >> 11
	t58a = ((t37*(4096-4017) + t58*799 + 2048) >> 12) - t37
	t59 = ((t36a*(4096-4017) + t59a*799 + 2048) >> 12) - t36a
	t60 = ((t35a*799 + t60a*(4017-4096) + 2048) >> 12) + t60a
	t61a = ((t34*799 + t61*(4017-4096) + 2048) >> 12) + t61
	t32 = clip_(t32a + t39a)
	t33a = clip_(t33 + t38)
	t34 = clip_(t34a + t37a)
	t35a = clip_(t35 + t36)
	t36a = clip_(t35 - t36)
	t37 = clip_(t34a - t37a)
	t38a = clip_(t33 - t38)
	t39 = clip_(t32a - t39a)
	t40 = clip_(t47a - t40a)
	t41a = clip_(t46 - t41)
	t42 = clip_(t45a - t42a)
	t43a = clip_(t44 - t43)
	t44a = clip_(t44 + t43)
	t45 = clip_(t45a + t42a)
	t46a = clip_(t46 + t41)
	t47 = clip_(t47a + t40a)
	t48 = clip_(t48a + t55a)
	t49a = clip_(t49 + t54)
	t50 = clip_(t50a + t53a)
	t51a = clip_(t51 + t52)
	t52a = clip_(t51 - t52)
	t53 = clip_(t50a - t53a)
	t54a = clip_(t49 - t54)
	t55 = clip_(t48a - t55a)
	t56 = clip_(t63a - t56a)
	t57a = clip_(t62 - t57)
	t58 = clip_(t61a - t58a)
	t59a = clip_(t60 - t59)
	t60a = clip_(t60 + t59)
	t61 = clip_(t61a + t58a)
	t62a = clip_(t62 + t57)
	t63 = clip_(t63a + t56a)
	t36 = ((t36a*(4096-3784) + t59a*1567 + 2048) >> 12) - t36a
	t37a = ((t37*(4096-3784) + t58*1567 + 2048) >> 12) - t37
	t38 = ((t38a*(4096-3784) + t57a*1567 + 2048) >> 12) - t38a
	t39a = ((t39*(4096-3784) + t56*1567 + 2048) >> 12) - t39
	t40a = ((t40*-1567 + t55*(4096-3784) + 2048) >> 12) - t55
	t41 = ((t41a*-1567 + t54a*(4096-3784) + 2048) >> 12) - t54a
	t42a = ((t42*-1567 + t53*(4096-3784) + 2048) >> 12) - t53
	t43 = ((t43a*-1567 + t52a*(4096-3784) + 2048) >> 12) - t52a
	t52 = ((t43a*(4096-3784) + t52a*1567 + 2048) >> 12) - t43a
	t53a = ((t42*(4096-3784) + t53*1567 + 2048) >> 12) - t42
	t54 = ((t41a*(4096-3784) + t54a*1567 + 2048) >> 12) - t41a
	t55a = ((t40*(4096-3784) + t55*1567 + 2048) >> 12) - t40
	t56a = ((t39*1567 + t56*(3784-4096) + 2048) >> 12) + t56
	t57 = ((t38a*1567 + t57a*(3784-4096) + 2048) >> 12) + t57a
	t58a = ((t37*1567 + t58*(3784-4096) + 2048) >> 12) + t58
	t59 = ((t36a*1567 + t59a*(3784-4096) + 2048) >> 12) + t59a
	t32a = clip_(t32 + t47)
	t33 = clip_(t33a + t46a)
	t34a = clip_(t34 + t45)
	t35 = clip_(t35a + t44a)
	t36a = clip_(t36 + t43)
	t37 = clip_(t37a + t42a)
	t38a = clip_(t38 + t41)
	t39 = clip_(t39a + t40a)
	t40 = clip_(t39a - t40a)
	t41a = clip_(t38 - t41)
	t42 = clip_(t37a - t42a)
	t43a = clip_(t36 - t43)
	t44 = clip_(t35a - t44a)
	t45a = clip_(t34 - t45)
	t46 = clip_(t33a - t46a)
	t47a = clip_(t32 - t47)
	t48a = clip_(t63 - t48)
	t49 = clip_(t62a - t49a)
	t50a = clip_(t61 - t50)
	t51 = clip_(t60a - t51a)
	t52a = clip_(t59 - t52)
	t53 = clip_(t58a - t53a)
	t54a = clip_(t57 - t54)
	t55 = clip_(t56a - t55a)
	t56 = clip_(t56a + t55a)
	t57a = clip_(t57 + t54)
	t58 = clip_(t58a + t53a)
	t59a = clip_(t59 + t52)
	t60 = clip_(t60a + t51a)
	t61a = clip_(t61 + t50)
	t62 = clip_(t62a + t49a)
	t63a = clip_(t63 + t48)
	t40a = ((t55-t40)*181 + 128) >> 8
	t41 = ((t54a-t41a)*181 + 128) >> 8
	t42a = ((t53-t42)*181 + 128) >> 8
	t43 = ((t52a-t43a)*181 + 128) >> 8
	t44a = ((t51-t44)*181 + 128) >> 8
	t45 = ((t50a-t45a)*181 + 128) >> 8
	t46a = ((t49-t46)*181 + 128) >> 8
	t47 = ((t48a-t47a)*181 + 128) >> 8
	t48 = ((t47a+t48a)*181 + 128) >> 8
	t49a = ((t46+t49)*181 + 128) >> 8
	t50 = ((t45a+t50a)*181 + 128) >> 8
	t51a = ((t44+t51)*181 + 128) >> 8
	t52 = ((t43a+t52a)*181 + 128) >> 8
	t53a = ((t42+t53)*181 + 128) >> 8
	t54 = ((t41a+t54a)*181 + 128) >> 8
	t55a = ((t40+t55)*181 + 128) >> 8
	t0 := c[cOff+0*stride]
	t1 := c[cOff+2*stride]
	t2 := c[cOff+4*stride]
	t3 := c[cOff+6*stride]
	t4 := c[cOff+8*stride]
	t5 := c[cOff+10*stride]
	t6 := c[cOff+12*stride]
	t7 := c[cOff+14*stride]
	t8 := c[cOff+16*stride]
	t9 := c[cOff+18*stride]
	t10 := c[cOff+20*stride]
	t11 := c[cOff+22*stride]
	t12 := c[cOff+24*stride]
	t13 := c[cOff+26*stride]
	t14 := c[cOff+28*stride]
	t15 := c[cOff+30*stride]
	t16 := c[cOff+32*stride]
	t17 := c[cOff+34*stride]
	t18 := c[cOff+36*stride]
	t19 := c[cOff+38*stride]
	t20 := c[cOff+40*stride]
	t21 := c[cOff+42*stride]
	t22 := c[cOff+44*stride]
	t23 := c[cOff+46*stride]
	t24 := c[cOff+48*stride]
	t25 := c[cOff+50*stride]
	t26 := c[cOff+52*stride]
	t27 := c[cOff+54*stride]
	t28 := c[cOff+56*stride]
	t29 := c[cOff+58*stride]
	t30 := c[cOff+60*stride]
	t31 := c[cOff+62*stride]
	c[cOff+0*stride] = clip_(t0 + t63a)
	c[cOff+1*stride] = clip_(t1 + t62)
	c[cOff+2*stride] = clip_(t2 + t61a)
	c[cOff+3*stride] = clip_(t3 + t60)
	c[cOff+4*stride] = clip_(t4 + t59a)
	c[cOff+5*stride] = clip_(t5 + t58)
	c[cOff+6*stride] = clip_(t6 + t57a)
	c[cOff+7*stride] = clip_(t7 + t56)
	c[cOff+8*stride] = clip_(t8 + t55a)
	c[cOff+9*stride] = clip_(t9 + t54)
	c[cOff+10*stride] = clip_(t10 + t53a)
	c[cOff+11*stride] = clip_(t11 + t52)
	c[cOff+12*stride] = clip_(t12 + t51a)
	c[cOff+13*stride] = clip_(t13 + t50)
	c[cOff+14*stride] = clip_(t14 + t49a)
	c[cOff+15*stride] = clip_(t15 + t48)
	c[cOff+16*stride] = clip_(t16 + t47)
	c[cOff+17*stride] = clip_(t17 + t46a)
	c[cOff+18*stride] = clip_(t18 + t45)
	c[cOff+19*stride] = clip_(t19 + t44a)
	c[cOff+20*stride] = clip_(t20 + t43)
	c[cOff+21*stride] = clip_(t21 + t42a)
	c[cOff+22*stride] = clip_(t22 + t41)
	c[cOff+23*stride] = clip_(t23 + t40a)
	c[cOff+24*stride] = clip_(t24 + t39)
	c[cOff+25*stride] = clip_(t25 + t38a)
	c[cOff+26*stride] = clip_(t26 + t37)
	c[cOff+27*stride] = clip_(t27 + t36a)
	c[cOff+28*stride] = clip_(t28 + t35)
	c[cOff+29*stride] = clip_(t29 + t34a)
	c[cOff+30*stride] = clip_(t30 + t33)
	c[cOff+31*stride] = clip_(t31 + t32a)
	c[cOff+32*stride] = clip_(t31 - t32a)
	c[cOff+33*stride] = clip_(t30 - t33)
	c[cOff+34*stride] = clip_(t29 - t34a)
	c[cOff+35*stride] = clip_(t28 - t35)
	c[cOff+36*stride] = clip_(t27 - t36a)
	c[cOff+37*stride] = clip_(t26 - t37)
	c[cOff+38*stride] = clip_(t25 - t38a)
	c[cOff+39*stride] = clip_(t24 - t39)
	c[cOff+40*stride] = clip_(t23 - t40a)
	c[cOff+41*stride] = clip_(t22 - t41)
	c[cOff+42*stride] = clip_(t21 - t42a)
	c[cOff+43*stride] = clip_(t20 - t43)
	c[cOff+44*stride] = clip_(t19 - t44a)
	c[cOff+45*stride] = clip_(t18 - t45)
	c[cOff+46*stride] = clip_(t17 - t46a)
	c[cOff+47*stride] = clip_(t16 - t47)
	c[cOff+48*stride] = clip_(t15 - t48)
	c[cOff+49*stride] = clip_(t14 - t49a)
	c[cOff+50*stride] = clip_(t13 - t50)
	c[cOff+51*stride] = clip_(t12 - t51a)
	c[cOff+52*stride] = clip_(t11 - t52)
	c[cOff+53*stride] = clip_(t10 - t53a)
	c[cOff+54*stride] = clip_(t9 - t54)
	c[cOff+55*stride] = clip_(t8 - t55a)
	c[cOff+56*stride] = clip_(t7 - t56)
	c[cOff+57*stride] = clip_(t6 - t57a)
	c[cOff+58*stride] = clip_(t5 - t58)
	c[cOff+59*stride] = clip_(t4 - t59a)
	c[cOff+60*stride] = clip_(t3 - t60)
	c[cOff+61*stride] = clip_(t2 - t61a)
	c[cOff+62*stride] = clip_(t1 - t62)
	c[cOff+63*stride] = clip_(t0 - t63a)
}

func invAdst41dInternal(in []int32, inOff, inS int, min, max int32, out []int32, outOff, outS int) {
	in0, in1 := in[inOff+0*inS], in[inOff+1*inS]
	in2, in3 := in[inOff+2*inS], in[inOff+3*inS]
	out[outOff+0*outS] = ((1321*in0 + (3803-4096)*in2 +
		(2482-4096)*in3 + (3344-4096)*in1 + 2048) >> 12) +
		in2 + in3 + in1
	out[outOff+1*outS] = (((2482-4096)*in0 - 1321*in2 -
		(3803-4096)*in3 + (3344-4096)*in1 + 2048) >> 12) +
		in0 - in3 + in1
	out[outOff+2*outS] = (209*(in0-in2+in3) + 128) >> 8
	out[outOff+3*outS] = (((3803-4096)*in0 + (2482-4096)*in2 -
		1321*in3 - (3344-4096)*in1 + 2048) >> 12) +
		in0 + in2 - in1
}

func invAdst81dInternal(in []int32, inOff, inS int, min, max int32, out []int32, outOff, outS int) {
	clip_ := func(v int32) int32 { return clip(v, min, max) }
	in0, in1 := in[inOff+0*inS], in[inOff+1*inS]
	in2, in3 := in[inOff+2*inS], in[inOff+3*inS]
	in4, in5 := in[inOff+4*inS], in[inOff+5*inS]
	in6, in7 := in[inOff+6*inS], in[inOff+7*inS]
	t0a := (((4076-4096)*in7 + 401*in0 + 2048) >> 12) + in7
	t1a := ((401*in7 - (4076-4096)*in0 + 2048) >> 12) - in0
	t2a := (((3612-4096)*in5 + 1931*in2 + 2048) >> 12) + in5
	t3a := ((1931*in5 - (3612-4096)*in2 + 2048) >> 12) - in2
	t4a := (1299*in3 + 1583*in4 + 1024) >> 11
	t5a := (1583*in3 - 1299*in4 + 1024) >> 11
	t6a := ((1189*in1 + (3920-4096)*in6 + 2048) >> 12) + in6
	t7a := (((3920-4096)*in1 - 1189*in6 + 2048) >> 12) + in1
	t0 := clip_(t0a + t4a)
	t1 := clip_(t1a + t5a)
	t2 := clip_(t2a + t6a)
	t3 := clip_(t3a + t7a)
	t4 := clip_(t0a - t4a)
	t5 := clip_(t1a - t5a)
	t6 := clip_(t2a - t6a)
	t7 := clip_(t3a - t7a)
	t4a = (((3784-4096)*t4 + 1567*t5 + 2048) >> 12) + t4
	t5a = ((1567*t4 - (3784-4096)*t5 + 2048) >> 12) - t5
	t6a = (((3784-4096)*t7 - 1567*t6 + 2048) >> 12) + t7
	t7a = ((1567*t7 + (3784-4096)*t6 + 2048) >> 12) + t6
	out[outOff+0*outS] = clip_(t0 + t2)
	out[outOff+7*outS] = -clip_(t1 + t3)
	t2 = clip_(t0 - t2)
	t3 = clip_(t1 - t3)
	out[outOff+1*outS] = -clip_(t4a + t6a)
	out[outOff+6*outS] = clip_(t5a + t7a)
	t6 = clip_(t4a - t6a)
	t7 = clip_(t5a - t7a)
	out[outOff+3*outS] = -(((t2+t3)*181 + 128) >> 8)
	out[outOff+4*outS] = ((t2-t3)*181 + 128) >> 8
	out[outOff+2*outS] = ((t6+t7)*181 + 128) >> 8
	out[outOff+5*outS] = -(((t6-t7)*181 + 128) >> 8)
}

func invAdst161dInternal(in []int32, inOff, inS int, min, max int32, out []int32, outOff, outS int) {
	clip_ := func(v int32) int32 { return clip(v, min, max) }
	in0, in1 := in[inOff+0*inS], in[inOff+1*inS]
	in2, in3 := in[inOff+2*inS], in[inOff+3*inS]
	in4, in5 := in[inOff+4*inS], in[inOff+5*inS]
	in6, in7 := in[inOff+6*inS], in[inOff+7*inS]
	in8, in9 := in[inOff+8*inS], in[inOff+9*inS]
	in10, in11 := in[inOff+10*inS], in[inOff+11*inS]
	in12, in13 := in[inOff+12*inS], in[inOff+13*inS]
	in14, in15 := in[inOff+14*inS], in[inOff+15*inS]
	t0 := ((in15*(4091-4096) + in0*201 + 2048) >> 12) + in15
	t1 := ((in15*201 - in0*(4091-4096) + 2048) >> 12) - in0
	t2 := ((in13*(3973-4096) + in2*995 + 2048) >> 12) + in13
	t3 := ((in13*995 - in2*(3973-4096) + 2048) >> 12) - in2
	t4 := ((in11*(3703-4096) + in4*1751 + 2048) >> 12) + in11
	t5 := ((in11*1751 - in4*(3703-4096) + 2048) >> 12) - in4
	t6 := (in9*1645 + in6*1220 + 1024) >> 11
	t7 := (in9*1220 - in6*1645 + 1024) >> 11
	t8 := ((in7*2751 + in8*(3035-4096) + 2048) >> 12) + in8
	t9 := ((in7*(3035-4096) - in8*2751 + 2048) >> 12) + in7
	t10 := ((in5*2106 + in10*(3513-4096) + 2048) >> 12) + in10
	t11 := ((in5*(3513-4096) - in10*2106 + 2048) >> 12) + in5
	t12 := ((in3*1380 + in12*(3857-4096) + 2048) >> 12) + in12
	t13 := ((in3*(3857-4096) - in12*1380 + 2048) >> 12) + in3
	t14 := ((in1*601 + in14*(4052-4096) + 2048) >> 12) + in14
	t15 := ((in1*(4052-4096) - in14*601 + 2048) >> 12) + in1
	t0a := clip_(t0 + t8)
	t1a := clip_(t1 + t9)
	t2a := clip_(t2 + t10)
	t3a := clip_(t3 + t11)
	t4a := clip_(t4 + t12)
	t5a := clip_(t5 + t13)
	t6a := clip_(t6 + t14)
	t7a := clip_(t7 + t15)
	t8a := clip_(t0 - t8)
	t9a := clip_(t1 - t9)
	t10a := clip_(t2 - t10)
	t11a := clip_(t3 - t11)
	t12a := clip_(t4 - t12)
	t13a := clip_(t5 - t13)
	t14a := clip_(t6 - t14)
	t15a := clip_(t7 - t15)
	t8 = ((t8a*(4017-4096) + t9a*799 + 2048) >> 12) + t8a
	t9 = ((t8a*799 - t9a*(4017-4096) + 2048) >> 12) - t9a
	t10 = ((t10a*2276 + t11a*(3406-4096) + 2048) >> 12) + t11a
	t11 = ((t10a*(3406-4096) - t11a*2276 + 2048) >> 12) + t10a
	t12 = ((t13a*(4017-4096) - t12a*799 + 2048) >> 12) + t13a
	t13 = ((t13a*799 + t12a*(4017-4096) + 2048) >> 12) + t12a
	t14 = ((t15a*2276 - t14a*(3406-4096) + 2048) >> 12) - t14a
	t15 = ((t15a*(3406-4096) + t14a*2276 + 2048) >> 12) + t15a
	t0 = clip_(t0a + t4a)
	t1 = clip_(t1a + t5a)
	t2 = clip_(t2a + t6a)
	t3 = clip_(t3a + t7a)
	t4 = clip_(t0a - t4a)
	t5 = clip_(t1a - t5a)
	t6 = clip_(t2a - t6a)
	t7 = clip_(t3a - t7a)
	t8a = clip_(t8 + t12)
	t9a = clip_(t9 + t13)
	t10a = clip_(t10 + t14)
	t11a = clip_(t11 + t15)
	t12a = clip_(t8 - t12)
	t13a = clip_(t9 - t13)
	t14a = clip_(t10 - t14)
	t15a = clip_(t11 - t15)
	t4a = ((t4*(3784-4096) + t5*1567 + 2048) >> 12) + t4
	t5a = ((t4*1567 - t5*(3784-4096) + 2048) >> 12) - t5
	t6a = ((t7*(3784-4096) - t6*1567 + 2048) >> 12) + t7
	t7a = ((t7*1567 + t6*(3784-4096) + 2048) >> 12) + t6
	t12 = ((t12a*(3784-4096) + t13a*1567 + 2048) >> 12) + t12a
	t13 = ((t12a*1567 - t13a*(3784-4096) + 2048) >> 12) - t13a
	t14 = ((t15a*(3784-4096) - t14a*1567 + 2048) >> 12) + t15a
	t15 = ((t15a*1567 + t14a*(3784-4096) + 2048) >> 12) + t14a
	out[outOff+0*outS] = clip_(t0 + t2)
	out[outOff+15*outS] = -clip_(t1 + t3)
	t2a = clip_(t0 - t2)
	t3a = clip_(t1 - t3)
	out[outOff+3*outS] = -clip_(t4a + t6a)
	out[outOff+12*outS] = clip_(t5a + t7a)
	t6 = clip_(t4a - t6a)
	t7 = clip_(t5a - t7a)
	out[outOff+1*outS] = -clip_(t8a + t10a)
	out[outOff+14*outS] = clip_(t9a + t11a)
	t10 = clip_(t8a - t10a)
	t11 = clip_(t9a - t11a)
	out[outOff+2*outS] = clip_(t12 + t14)
	out[outOff+13*outS] = -clip_(t13 + t15)
	t14a = clip_(t12 - t14)
	t15a = clip_(t13 - t15)
	out[outOff+7*outS] = -(((t2a+t3a)*181 + 128) >> 8)
	out[outOff+8*outS] = ((t2a-t3a)*181 + 128) >> 8
	out[outOff+4*outS] = ((t6+t7)*181 + 128) >> 8
	out[outOff+11*outS] = -(((t6-t7)*181 + 128) >> 8)
	out[outOff+6*outS] = ((t10+t11)*181 + 128) >> 8
	out[outOff+9*outS] = -(((t10-t11)*181 + 128) >> 8)
	out[outOff+5*outS] = -(((t14a+t15a)*181 + 128) >> 8)
	out[outOff+10*outS] = ((t14a-t15a)*181 + 128) >> 8
}

func invIdentity41d(c []int32, cOff, stride int, min, max int32) {
	for i := range 4 {
		in := c[cOff+stride*i]
		c[cOff+stride*i] = in + ((in*1697 + 2048) >> 12)
	}
}

func invIdentity81d(c []int32, cOff, stride int, min, max int32) {
	for i := range 8 {
		c[cOff+stride*i] *= 2
	}
}

func invIdentity161d(c []int32, cOff, stride int, min, max int32) {
	for i := range 16 {
		in := c[cOff+stride*i]
		c[cOff+stride*i] = 2*in + ((in*1697 + 1024) >> 11)
	}
}

func invIdentity321d(c []int32, cOff, stride int, min, max int32) {
	for i := range 32 {
		c[cOff+stride*i] *= 4
	}
}

var itxIdentityParams = [4][4]int32{
	{1, 1697, 2048, 12},
	{2, 0, 0, 12},
	{2, 1697, 1024, 11},
	{4, 0, 0, 12},
}

func invAdst41d(c []int32, cOff, stride int, min, max int32) {
	invAdst41dInternal(c, cOff, stride, min, max, c, cOff, stride)
}

func invFlipadst41d(c []int32, cOff, stride int, min, max int32) {
	invAdst41dInternal(c, cOff, stride, min, max, c, cOff+3*stride, -stride)
}

func invAdst81d(c []int32, cOff, stride int, min, max int32) {
	invAdst81dInternal(c, cOff, stride, min, max, c, cOff, stride)
}

func invFlipadst81d(c []int32, cOff, stride int, min, max int32) {
	invAdst81dInternal(c, cOff, stride, min, max, c, cOff+7*stride, -stride)
}

func invAdst161d(c []int32, cOff, stride int, min, max int32) {
	invAdst161dInternal(c, cOff, stride, min, max, c, cOff, stride)
}

func invFlipadst161d(c []int32, cOff, stride int, min, max int32) {
	invAdst161dInternal(c, cOff, stride, min, max, c, cOff+15*stride, -stride)
}

func invWht41d(c []int32, cOff, stride int) {
	in0, in1 := c[cOff+0*stride], c[cOff+1*stride]
	in2, in3 := c[cOff+2*stride], c[cOff+3*stride]

	t0 := in0 + in1
	t2 := in2 - in3
	t4 := (t0 - t2) >> 1
	t3 := t4 - in3
	t1 := t4 - in1

	c[cOff+0*stride] = t0 - t3
	c[cOff+1*stride] = t3
	c[cOff+2*stride] = t1
	c[cOff+3*stride] = t2 + t1
}

const (
	tx1dDct = iota
	tx1dAdst
	tx1dIdentity
	tx1dFlipadst
	nTx1dTypes
)

type itx1dFn func(c []int32, cOff, stride int, min, max int32)

var tx1dFns = [nTxSizes][nTx1dTypes]itx1dFn{
	tx4x4: {
		tx1dDct:      invDct41d,
		tx1dAdst:     invAdst41d,
		tx1dFlipadst: invFlipadst41d,
		tx1dIdentity: invIdentity41d,
	},
	tx8x8: {
		tx1dDct:      invDct81d,
		tx1dAdst:     invAdst81d,
		tx1dFlipadst: invFlipadst81d,
		tx1dIdentity: invIdentity81d,
	},
	tx16x16: {
		tx1dDct:      invDct161d,
		tx1dAdst:     invAdst161d,
		tx1dFlipadst: invFlipadst161d,
		tx1dIdentity: invIdentity161d,
	},
	tx32x32: {
		tx1dDct:      invDct321d,
		tx1dIdentity: invIdentity321d,
	},
	tx64x64: {
		tx1dDct: invDct641d,
	},
}

var tx1dTypes = [nTxTypes][2]uint8{
	dctDct:           {tx1dDct, tx1dDct},
	adstDct:          {tx1dAdst, tx1dDct},
	dctAdst:          {tx1dDct, tx1dAdst},
	adstAdst:         {tx1dAdst, tx1dAdst},
	flipadstDct:      {tx1dFlipadst, tx1dDct},
	dctFlipadst:      {tx1dDct, tx1dFlipadst},
	flipadstFlipadst: {tx1dFlipadst, tx1dFlipadst},
	adstFlipadst:     {tx1dAdst, tx1dFlipadst},
	flipadstAdst:     {tx1dFlipadst, tx1dAdst},
	idtx:             {tx1dIdentity, tx1dIdentity},
	vDct:             {tx1dDct, tx1dIdentity},
	hDct:             {tx1dIdentity, tx1dDct},
	vAdst:            {tx1dAdst, tx1dIdentity},
	hAdst:            {tx1dIdentity, tx1dAdst},
	vFlipadst:        {tx1dFlipadst, tx1dIdentity},
	hFlipadst:        {tx1dIdentity, tx1dFlipadst},
}
