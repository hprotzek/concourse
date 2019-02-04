module Msgs exposing (Msg(..), NavIndex)

import Callback exposing (Callback)
import Effects
import Routes
import SubPage.Msgs
import TopBar.Msgs


type alias NavIndex =
    Int


type Msg
    = RouteChanged Routes.Route
    | SubMsg NavIndex SubPage.Msgs.Msg
    | TopMsg NavIndex TopBar.Msgs.Msg
    | NewUrl String
    | ModifyUrl Routes.Route
    | TokenReceived (Maybe String)
    | Callback Effects.LayoutDispatch Callback



-- NewUrl must be a String because of the subscriptions, and nasty type-contravariance. :(
